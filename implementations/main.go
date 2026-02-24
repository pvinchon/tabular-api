package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
)

// ──────────────────────────────────────────────
// Configuration
// ──────────────────────────────────────────────

type firebaseConfig struct {
	ProjectID        string
	APIKey           string
	AuthDomain       string
	AuthEmulatorHost string // e.g. "firebase-emulator:9099"; empty = production
}

func loadFirebaseConfig() firebaseConfig {
	cfg := firebaseConfig{
		ProjectID:  os.Getenv("FIREBASE_PROJECT_ID"),
		APIKey:     os.Getenv("FIREBASE_API_KEY"),
		AuthDomain: os.Getenv("FIREBASE_AUTH_DOMAIN"),
	}
	missing := []string{}
	if cfg.ProjectID == "" {
		missing = append(missing, "FIREBASE_PROJECT_ID")
	}
	if cfg.APIKey == "" {
		missing = append(missing, "FIREBASE_API_KEY")
	}
	if cfg.AuthDomain == "" {
		missing = append(missing, "FIREBASE_AUTH_DOMAIN")
	}
	if len(missing) > 0 {
		slog.Error("missing required environment variables", "vars", strings.Join(missing, ", "))
		os.Exit(1)
	}

	cfg.AuthEmulatorHost = os.Getenv("FIREBASE_AUTH_EMULATOR_HOST")
	if cfg.AuthEmulatorHost != "" {
		slog.Warn("running with Firebase Auth emulator", "host", cfg.AuthEmulatorHost)
	}
	return cfg
}

// ──────────────────────────────────────────────
// Public Key Cache (Google's signing keys)
// ──────────────────────────────────────────────

const googleCertsURL = "https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com"

type publicKeyCache struct {
	mu     sync.RWMutex
	keys   map[string]*rsa.PublicKey
	expiry time.Time
}

var keyCache = &publicKeyCache{}

func (c *publicKeyCache) getKey(kid string) (*rsa.PublicKey, error) {
	c.mu.RLock()
	if time.Now().Before(c.expiry) {
		if key, ok := c.keys[kid]; ok {
			c.mu.RUnlock()
			return key, nil
		}
		c.mu.RUnlock()
		return nil, fmt.Errorf("key ID %q not found in cache", kid)
	}
	c.mu.RUnlock()

	// Cache expired or empty — refresh
	if err := c.refresh(); err != nil {
		return nil, fmt.Errorf("failed to refresh public keys: %w", err)
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	if key, ok := c.keys[kid]; ok {
		return key, nil
	}
	return nil, fmt.Errorf("key ID %q not found after refresh", kid)
}

func (c *publicKeyCache) refresh() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if time.Now().Before(c.expiry) {
		return nil
	}

	resp, err := http.Get(googleCertsURL)
	if err != nil {
		return fmt.Errorf("fetching Google certs: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading Google certs response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Google certs returned status %d", resp.StatusCode)
	}

	var certMap map[string]string
	if err := json.Unmarshal(body, &certMap); err != nil {
		return fmt.Errorf("parsing Google certs JSON: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(certMap))
	for kid, certPEM := range certMap {
		block, _ := pem.Decode([]byte(certPEM))
		if block == nil {
			return fmt.Errorf("failed to decode PEM for key %q", kid)
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return fmt.Errorf("parsing certificate for key %q: %w", kid, err)
		}
		rsaKey, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return fmt.Errorf("key %q is not RSA", kid)
		}
		keys[kid] = rsaKey
	}

	// Parse max-age from Cache-Control header
	maxAge := 3600 // default 1 hour
	if cc := resp.Header.Get("Cache-Control"); cc != "" {
		for _, directive := range strings.Split(cc, ",") {
			directive = strings.TrimSpace(directive)
			if strings.HasPrefix(directive, "max-age=") {
				if v, err := strconv.Atoi(strings.TrimPrefix(directive, "max-age=")); err == nil {
					maxAge = v
				}
			}
		}
	}

	c.keys = keys
	c.expiry = time.Now().Add(time.Duration(maxAge) * time.Second)
	slog.Info("refreshed Google public keys", "count", len(keys), "expires_in_seconds", maxAge)
	return nil
}

// ──────────────────────────────────────────────
// JWT Verification
// ──────────────────────────────────────────────

type userClaims struct {
	UID     string `json:"uid"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

type firebaseClaims struct {
	jwt.RegisteredClaims
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// verifyEmulatorToken parses an unsigned emulator token without
// signature verification. The emulator uses alg:"none".
func verifyEmulatorToken(tokenString string, projectID string) (*userClaims, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{"none", "RS256"}),
		jwt.WithoutClaimsValidation(),
	)

	token, _, err := parser.ParseUnverified(tokenString, &firebaseClaims{})
	if err != nil {
		return nil, fmt.Errorf("parsing emulator token: %w", err)
	}

	claims, ok := token.Claims.(*firebaseClaims)
	if !ok {
		return nil, fmt.Errorf("invalid emulator token claims")
	}

	if claims.Subject == "" {
		return nil, fmt.Errorf("emulator token subject (uid) is empty")
	}

	return &userClaims{
		UID:     claims.Subject,
		Email:   claims.Email,
		Name:    claims.Name,
		Picture: claims.Picture,
	}, nil
}

func verifyIDToken(tokenString string, projectID string) (*userClaims, error) {
	// Parse without verification first to get the key ID
	token, parts, err := jwt.NewParser().ParseUnverified(tokenString, &firebaseClaims{})
	if err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}
	_ = parts

	// Check algorithm
	if token.Method.Alg() != "RS256" {
		return nil, fmt.Errorf("unexpected signing algorithm: %s", token.Method.Alg())
	}

	// Get the key ID
	kid, ok := token.Header["kid"].(string)
	if !ok || kid == "" {
		return nil, fmt.Errorf("missing kid in token header")
	}

	// Fetch the public key
	pubKey, err := keyCache.getKey(kid)
	if err != nil {
		return nil, err
	}

	// Parse and verify the token with the public key
	expectedIssuer := "https://securetoken.google.com/" + projectID
	verifiedToken, err := jwt.ParseWithClaims(tokenString, &firebaseClaims{}, func(t *jwt.Token) (interface{}, error) {
		return pubKey, nil
	},
		jwt.WithValidMethods([]string{"RS256"}),
	)
	if err != nil {
		return nil, fmt.Errorf("token verification failed: %w", err)
	}

	claims, ok := verifiedToken.Claims.(*firebaseClaims)
	if !ok || !verifiedToken.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	if claims.Subject == "" {
		return nil, fmt.Errorf("token subject (uid) is empty")
	}

	// Verify issuer
	if claims.Issuer != expectedIssuer {
		return nil, fmt.Errorf("invalid issuer: got %q, want %q", claims.Issuer, expectedIssuer)
	}

	// Verify audience
	foundAud := false
	for _, aud := range claims.Audience {
		if aud == projectID {
			foundAud = true
			break
		}
	}
	if !foundAud {
		return nil, fmt.Errorf("invalid audience: %v does not contain %q", claims.Audience, projectID)
	}

	return &userClaims{
		UID:     claims.Subject,
		Email:   claims.Email,
		Name:    claims.Name,
		Picture: claims.Picture,
	}, nil
}

// ──────────────────────────────────────────────
// JSON Helpers
// ──────────────────────────────────────────────

type errorEnvelope struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorEnvelope{
		Error: errorDetail{Code: code, Message: message},
	})
}

// ──────────────────────────────────────────────
// HTML Pages
// ──────────────────────────────────────────────

const firebaseSDKVersion = "11.3.0"

// emulatorConnectSnippet returns JS to connect to the Firebase Auth
// emulator after getAuth(). Empty string if not using emulator.
func emulatorConnectSnippet(cfg firebaseConfig) string {
	if cfg.AuthEmulatorHost == "" {
		return ""
	}
	return "\n        connectAuthEmulator(auth, \"http://" + cfg.AuthEmulatorHost + "\", { disableWarnings: true });\n"
}

func homePage(cfg firebaseConfig) string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Hello, World!</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; max-width: 600px; margin: 40px auto; padding: 0 20px; }
        .auth-section { margin-top: 20px; padding: 20px; border: 1px solid #ddd; border-radius: 8px; }
        .user-info { display: flex; align-items: center; gap: 12px; }
        .btn { padding: 10px 24px; font-size: 16px; border: none; border-radius: 6px; cursor: pointer; }
        .btn-signin { background: #4285f4; color: white; }
        .btn-signin:hover { background: #3367d6; }
        .btn-signout { background: #f44336; color: white; }
        .btn-signout:hover { background: #d32f2f; }
        .btn-profile { background: #4caf50; color: white; text-decoration: none; display: inline-block; }
        .btn-profile:hover { background: #388e3c; }
        #loading { color: #666; }
        #error-msg { color: #f44336; margin-top: 10px; display: none; }
    </style>
</head>
<body>
    <h1>Hello, World!</h1>

    <div class="auth-section">
        <div id="loading">Loading...</div>
        <div id="signed-out" style="display:none">
            <p>You are not signed in.</p>
            <button class="btn btn-signin" id="signin-btn">Sign in with Google</button>
        </div>
        <div id="signed-in" style="display:none">
            <div class="user-info">
                <span>Welcome, <strong id="user-name"></strong></span>
            </div>
            <div style="margin-top: 12px; display: flex; gap: 8px;">
                <a href="/profile" class="btn btn-profile">View Profile</a>
                <button class="btn btn-signout" id="signout-btn">Sign out</button>
            </div>
        </div>
        <div id="error-msg"></div>
    </div>

    <script type="module">
        import { initializeApp } from "https://www.gstatic.com/firebasejs/` + firebaseSDKVersion + `/firebase-app.js";
        import { getAuth, connectAuthEmulator, signInWithPopup, GoogleAuthProvider, onAuthStateChanged, signOut } from "https://www.gstatic.com/firebasejs/` + firebaseSDKVersion + `/firebase-auth.js";

        const firebaseConfig = {
            apiKey: "` + cfg.APIKey + `",
            authDomain: "` + cfg.AuthDomain + `",
            projectId: "` + cfg.ProjectID + `"
        };

        const app = initializeApp(firebaseConfig);
        const auth = getAuth(app);
` + emulatorConnectSnippet(cfg) + `        const provider = new GoogleAuthProvider();

        const loadingEl = document.getElementById("loading");
        const signedOutEl = document.getElementById("signed-out");
        const signedInEl = document.getElementById("signed-in");
        const userNameEl = document.getElementById("user-name");
        const errorEl = document.getElementById("error-msg");

        onAuthStateChanged(auth, (user) => {
            loadingEl.style.display = "none";
            if (user) {
                userNameEl.textContent = user.displayName || user.email;
                signedInEl.style.display = "block";
                signedOutEl.style.display = "none";
            } else {
                signedInEl.style.display = "none";
                signedOutEl.style.display = "block";
            }
        });

        document.getElementById("signin-btn").addEventListener("click", async () => {
            try {
                await signInWithPopup(auth, provider);
            } catch (err) {
                if (err.code === "auth/popup-closed-by-user" || err.code === "auth/cancelled-popup-request") {
                    return; // User cancelled — not an error
                }
                errorEl.textContent = "Sign-in failed: " + err.message;
                errorEl.style.display = "block";
            }
        });

        document.getElementById("signout-btn").addEventListener("click", async () => {
            try {
                await signOut(auth);
            } catch (err) {
                errorEl.textContent = "Sign-out failed: " + err.message;
                errorEl.style.display = "block";
            }
        });
    </script>
</body>
</html>`
}

func profilePage(cfg firebaseConfig) string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Profile</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; max-width: 600px; margin: 40px auto; padding: 0 20px; }
        .profile-card { padding: 24px; border: 1px solid #ddd; border-radius: 8px; }
        .profile-header { display: flex; align-items: center; gap: 16px; margin-bottom: 16px; }
        .profile-pic { width: 80px; height: 80px; border-radius: 50%; object-fit: cover; background: #e0e0e0; }
        .placeholder-pic { width: 80px; height: 80px; border-radius: 50%; background: #9e9e9e; display: flex; align-items: center; justify-content: center; color: white; font-size: 32px; }
        .profile-details { margin-top: 12px; }
        .profile-details dt { font-weight: bold; color: #555; margin-top: 8px; }
        .profile-details dd { margin-left: 0; }
        .btn { padding: 10px 24px; font-size: 16px; border: none; border-radius: 6px; cursor: pointer; }
        .btn-signout { background: #f44336; color: white; margin-top: 16px; }
        .btn-signout:hover { background: #d32f2f; }
        .btn-home { background: #2196f3; color: white; text-decoration: none; display: inline-block; margin-top: 16px; margin-right: 8px; }
        .btn-home:hover { background: #1976d2; }
        #loading { color: #666; }
        #error-msg { color: #f44336; margin-top: 10px; display: none; }
    </style>
</head>
<body>
    <h1>Profile</h1>

    <div id="loading">Loading profile...</div>
    <div id="profile-card" class="profile-card" style="display:none">
        <div class="profile-header">
            <div id="pic-container"></div>
            <div>
                <h2 id="profile-name" style="margin:0"></h2>
                <p id="profile-email" style="margin:4px 0 0 0; color:#666"></p>
            </div>
        </div>
        <dl class="profile-details">
            <dt>User ID</dt>
            <dd id="profile-uid"></dd>
        </dl>
        <div>
            <a href="/" class="btn btn-home">Home</a>
            <button class="btn btn-signout" id="signout-btn">Sign out</button>
        </div>
    </div>
    <div id="error-msg"></div>

    <script type="module">
        import { initializeApp } from "https://www.gstatic.com/firebasejs/` + firebaseSDKVersion + `/firebase-app.js";
        import { getAuth, connectAuthEmulator, signInWithPopup, GoogleAuthProvider, onAuthStateChanged, signOut } from "https://www.gstatic.com/firebasejs/` + firebaseSDKVersion + `/firebase-auth.js";

        const firebaseConfig = {
            apiKey: "` + cfg.APIKey + `",
            authDomain: "` + cfg.AuthDomain + `",
            projectId: "` + cfg.ProjectID + `"
        };

        const app = initializeApp(firebaseConfig);
        const auth = getAuth(app);
` + emulatorConnectSnippet(cfg) + `        const provider = new GoogleAuthProvider();

        const loadingEl = document.getElementById("loading");
        const profileCard = document.getElementById("profile-card");
        const errorEl = document.getElementById("error-msg");

        onAuthStateChanged(auth, async (user) => {
            if (!user) {
                // Unauthenticated — auto-initiate sign-in (FR-009)
                loadingEl.textContent = "Redirecting to sign in...";
                try {
                    await signInWithPopup(auth, provider);
                } catch (err) {
                    if (err.code === "auth/popup-closed-by-user" || err.code === "auth/cancelled-popup-request") {
                        loadingEl.textContent = "Sign-in was cancelled. Please sign in to view your profile.";
                        return;
                    }
                    errorEl.textContent = "Sign-in failed: " + err.message;
                    errorEl.style.display = "block";
                    loadingEl.style.display = "none";
                }
                return;
            }

            // Authenticated — fetch profile from API
            try {
                const idToken = await user.getIdToken();
                const resp = await fetch("/api/me", {
                    headers: { "Authorization": "Bearer " + idToken }
                });

                if (!resp.ok) {
                    const errData = await resp.json();
                    throw new Error(errData.error?.message || "Failed to load profile");
                }

                const profile = await resp.json();
                document.getElementById("profile-name").textContent = profile.name || "Unknown";
                document.getElementById("profile-email").textContent = profile.email || "";
                document.getElementById("profile-uid").textContent = profile.uid || "";

                const picContainer = document.getElementById("pic-container");
                if (profile.picture) {
                    picContainer.innerHTML = '<img class="profile-pic" src="' + profile.picture + '" alt="Profile picture" referrerpolicy="no-referrer">';
                } else {
                    const initial = (profile.name || "?")[0].toUpperCase();
                    picContainer.innerHTML = '<div class="placeholder-pic">' + initial + '</div>';
                }

                loadingEl.style.display = "none";
                profileCard.style.display = "block";
            } catch (err) {
                errorEl.textContent = "Error loading profile: " + err.message;
                errorEl.style.display = "block";
                loadingEl.style.display = "none";
            }
        });

        document.getElementById("signout-btn").addEventListener("click", async () => {
            try {
                await signOut(auth);
                // onAuthStateChanged will fire and re-initiate sign-in
            } catch (err) {
                errorEl.textContent = "Sign-out failed: " + err.message;
                errorEl.style.display = "block";
            }
        });
    </script>
</body>
</html>`
}

// ──────────────────────────────────────────────
// Router Setup (extracted for testability)
// ──────────────────────────────────────────────

func newMux(cfg firebaseConfig) *http.ServeMux {
	mux := http.NewServeMux()

	// GET / — Home page with Hello, World! and auth UI
	homeHTML := homePage(cfg)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, homeHTML)
	})

	// GET /profile — Profile page
	profileHTML := profilePage(cfg)
	mux.HandleFunc("GET /profile", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, profileHTML)
	})

	// GET /api/me — Authenticated user profile (JSON)
	mux.HandleFunc("GET /api/me", func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Missing or invalid authentication token")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		var user *userClaims
		var err error
		if cfg.AuthEmulatorHost != "" {
			user, err = verifyEmulatorToken(tokenString, cfg.ProjectID)
		} else {
			user, err = verifyIDToken(tokenString, cfg.ProjectID)
		}
		if err != nil {
			slog.Warn("token verification failed", "error", err.Error())
			writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Missing or invalid authentication token")
			return
		}

		writeJSON(w, http.StatusOK, user)
	})

	// Catch-all 404
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	return mux
}

// ──────────────────────────────────────────────
// Main
// ──────────────────────────────────────────────

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		slog.Error("PORT environment variable is required but not set")
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := loadFirebaseConfig()

	mux := newMux(cfg)

	handler := loggingMiddleware(mux)

	addr := ":" + port
	slog.Info("server starting", "addr", addr)

	if err := http.ListenAndServe(addr, handler); err != nil {
		slog.Error("server failed", "error", err.Error())
		os.Exit(1)
	}
}

// ──────────────────────────────────────────────
// Logging Middleware
// ──────────────────────────────────────────────

type responseCapture struct {
	http.ResponseWriter
	status int
}

func (rc *responseCapture) WriteHeader(code int) {
	rc.status = code
	rc.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(next http.Handler) http.Handler {
	var counter uint64
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		counter++
		requestID := fmt.Sprintf("%d-%d", start.UnixNano(), counter)

		rc := &responseCapture{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rc, r)

		latency := time.Since(start)
		slog.Info("request",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", rc.status,
			"latency_ms", float64(latency.Microseconds())/1000.0,
		)
	})
}
