package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
)

// ── Test helpers ────────────────────────────────

const testProjectID = "test-project-123"

var testCfg = firebaseConfig{
	ProjectID:  testProjectID,
	APIKey:     "AIzaSyTestKey",
	AuthDomain: "test-project-123.firebaseapp.com",
}

func generateTestKey(t *testing.T, kid string) *rsa.PrivateKey {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}
	keyCache.mu.Lock()
	keyCache.keys = map[string]*rsa.PublicKey{kid: &privKey.PublicKey}
	keyCache.expiry = time.Now().Add(1 * time.Hour)
	keyCache.mu.Unlock()
	return privKey
}

func signToken(t *testing.T, privKey *rsa.PrivateKey, kid string, claims firebaseClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	s, err := token.SignedString(privKey)
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}
	return s
}

func validClaims() firebaseClaims {
	now := time.Now()
	return firebaseClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://securetoken.google.com/" + testProjectID,
			Audience:  jwt.ClaimStrings{testProjectID},
			Subject:   "user-uid-abc123",
			ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now.Add(-5 * time.Minute)),
		},
		Email:   "jane@example.com",
		Name:    "Jane Doe",
		Picture: "https://lh3.googleusercontent.com/a/photo",
	}
}

func newTestServer() *httptest.Server {
	return httptest.NewServer(newMux(testCfg))
}

// ── GET / ───────────────────────────────────────

func TestHomePage_Status200(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestHomePage_ContainsHelloWorld(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Hello, World!") {
		t.Error("body missing 'Hello, World!'")
	}
}

func TestHomePage_ContentType(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/html; charset=utf-8", ct)
	}
}

func TestHomePage_HasSignInButton(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Sign in with Google") {
		t.Error("missing 'Sign in with Google' button")
	}
}

func TestHomePage_HasSignOutButton(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Sign out") {
		t.Error("missing 'Sign out' button")
	}
}

func TestHomePage_HasProfileLink(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `href="/profile"`) {
		t.Error("missing link to /profile")
	}
}

func TestHomePage_HasFirebaseConfig(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	s := string(body)
	if !strings.Contains(s, testCfg.APIKey) {
		t.Error("missing Firebase API key")
	}
	if !strings.Contains(s, testCfg.AuthDomain) {
		t.Error("missing Firebase auth domain")
	}
	if !strings.Contains(s, testCfg.ProjectID) {
		t.Error("missing Firebase project ID")
	}
}

func TestHomePage_HasFirebaseSDK(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	s := string(body)
	if !strings.Contains(s, "firebase-app.js") {
		t.Error("missing firebase-app.js")
	}
	if !strings.Contains(s, "firebase-auth.js") {
		t.Error("missing firebase-auth.js")
	}
	if !strings.Contains(s, firebaseSDKVersion) {
		t.Errorf("missing pinned SDK version %s", firebaseSDKVersion)
	}
}

// ── GET /profile ────────────────────────────────

func TestProfilePage_Status200(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/profile")
	if err != nil {
		t.Fatalf("GET /profile: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestProfilePage_ContentType(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/profile")
	if err != nil {
		t.Fatalf("GET /profile: %v", err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/html; charset=utf-8", ct)
	}
}

func TestProfilePage_HasSignOutButton(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/profile")
	if err != nil {
		t.Fatalf("GET /profile: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Sign out") {
		t.Error("profile missing 'Sign out' button")
	}
}

func TestProfilePage_HasHomeLink(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/profile")
	if err != nil {
		t.Fatalf("GET /profile: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `href="/"`) {
		t.Error("profile missing link to home")
	}
}

func TestProfilePage_HasFirebaseConfig(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/profile")
	if err != nil {
		t.Fatalf("GET /profile: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	s := string(body)
	if !strings.Contains(s, testCfg.APIKey) {
		t.Error("profile missing Firebase API key")
	}
	if !strings.Contains(s, testCfg.ProjectID) {
		t.Error("profile missing Firebase project ID")
	}
}

func TestProfilePage_CallsAPIMe(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/profile")
	if err != nil {
		t.Fatalf("GET /profile: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `/api/me`) {
		t.Error("profile missing /api/me call")
	}
}

func TestProfilePage_AutoSignIn(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/profile")
	if err != nil {
		t.Fatalf("GET /profile: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "signInWithPopup") {
		t.Error("profile missing auto sign-in for unauthenticated users")
	}
}

// ── GET /api/me ─────────────────────────────────

func TestAPIMe_NoAuth_401(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/api/me")
	if err != nil {
		t.Fatalf("GET /api/me: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestAPIMe_NoAuth_ErrorEnvelope(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/api/me")
	if err != nil {
		t.Fatalf("GET /api/me: %v", err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var env errorEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Error.Code != "UNAUTHENTICATED" {
		t.Errorf("code = %q, want UNAUTHENTICATED", env.Error.Code)
	}
	if env.Error.Message == "" {
		t.Error("message is empty")
	}
}

func TestAPIMe_InvalidToken_401(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	req, _ := http.NewRequest("GET", srv.URL+"/api/me", nil)
	req.Header.Set("Authorization", "Bearer garbage")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/me: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestAPIMe_BasicAuth_401(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	req, _ := http.NewRequest("GET", srv.URL+"/api/me", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/me: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestAPIMe_ValidToken_200(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	kid := "key-valid"
	privKey := generateTestKey(t, kid)
	tok := signToken(t, privKey, kid, validClaims())
	req, _ := http.NewRequest("GET", srv.URL+"/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/me: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200; body = %s", resp.StatusCode, b)
	}
	var u userClaims
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if u.UID != "user-uid-abc123" {
		t.Errorf("uid = %q", u.UID)
	}
	if u.Email != "jane@example.com" {
		t.Errorf("email = %q", u.Email)
	}
	if u.Name != "Jane Doe" {
		t.Errorf("name = %q", u.Name)
	}
	if u.Picture != "https://lh3.googleusercontent.com/a/photo" {
		t.Errorf("picture = %q", u.Picture)
	}
}

func TestAPIMe_ValidToken_JSON(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	kid := "key-json"
	privKey := generateTestKey(t, kid)
	tok := signToken(t, privKey, kid, validClaims())
	req, _ := http.NewRequest("GET", srv.URL+"/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/me: %v", err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestAPIMe_NoPicture_EmptyString(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	kid := "key-nopic"
	privKey := generateTestKey(t, kid)
	c := validClaims()
	c.Picture = ""
	tok := signToken(t, privKey, kid, c)
	req, _ := http.NewRequest("GET", srv.URL+"/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/me: %v", err)
	}
	defer resp.Body.Close()
	var u userClaims
	json.NewDecoder(resp.Body).Decode(&u)
	if u.Picture != "" {
		t.Errorf("picture = %q, want empty", u.Picture)
	}
}

// ── verifyIDToken unit tests ────────────────────

func TestVerify_Valid(t *testing.T) {
	kid := "v-valid"
	pk := generateTestKey(t, kid)
	tok := signToken(t, pk, kid, validClaims())
	u, err := verifyIDToken(tok, testProjectID)
	if err != nil {
		t.Fatalf("verifyIDToken: %v", err)
	}
	if u.UID != "user-uid-abc123" {
		t.Errorf("uid = %q", u.UID)
	}
	if u.Email != "jane@example.com" {
		t.Errorf("email = %q", u.Email)
	}
	if u.Name != "Jane Doe" {
		t.Errorf("name = %q", u.Name)
	}
}

func TestVerify_Expired(t *testing.T) {
	kid := "v-expired"
	pk := generateTestKey(t, kid)
	c := validClaims()
	c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-1 * time.Hour))
	tok := signToken(t, pk, kid, c)
	_, err := verifyIDToken(tok, testProjectID)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestVerify_WrongIssuer(t *testing.T) {
	kid := "v-iss"
	pk := generateTestKey(t, kid)
	c := validClaims()
	c.Issuer = "https://securetoken.google.com/wrong-project"
	tok := signToken(t, pk, kid, c)
	_, err := verifyIDToken(tok, testProjectID)
	if err == nil {
		t.Error("expected error for wrong issuer")
	}
	if !strings.Contains(err.Error(), "issuer") {
		t.Errorf("error should mention issuer: %v", err)
	}
}

func TestVerify_WrongAudience(t *testing.T) {
	kid := "v-aud"
	pk := generateTestKey(t, kid)
	c := validClaims()
	c.Audience = jwt.ClaimStrings{"wrong-project"}
	tok := signToken(t, pk, kid, c)
	_, err := verifyIDToken(tok, testProjectID)
	if err == nil {
		t.Error("expected error for wrong audience")
	}
	if !strings.Contains(err.Error(), "audience") {
		t.Errorf("error should mention audience: %v", err)
	}
}

func TestVerify_EmptySubject(t *testing.T) {
	kid := "v-sub"
	pk := generateTestKey(t, kid)
	c := validClaims()
	c.Subject = ""
	tok := signToken(t, pk, kid, c)
	_, err := verifyIDToken(tok, testProjectID)
	if err == nil {
		t.Error("expected error for empty subject")
	}
}

func TestVerify_WrongKey(t *testing.T) {
	kid := "v-wrongkey"
	generateTestKey(t, kid)
	otherKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	tok := signToken(t, otherKey, kid, validClaims())
	_, err := verifyIDToken(tok, testProjectID)
	if err == nil {
		t.Error("expected error for wrong signing key")
	}
}

func TestVerify_UnknownKID(t *testing.T) {
	kid := "v-known"
	pk := generateTestKey(t, kid)
	tok := signToken(t, pk, "v-unknown", validClaims())
	_, err := verifyIDToken(tok, testProjectID)
	if err == nil {
		t.Error("expected error for unknown kid")
	}
}

// ── 404 catch-all ───────────────────────────────

func TestCatchAll_404(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/nonexistent")
	if err != nil {
		t.Fatalf("GET /nonexistent: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestCatchAll_EmptyBody(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/nonexistent")
	if err != nil {
		t.Fatalf("GET /nonexistent: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if len(body) != 0 {
		t.Errorf("404 body should be empty, got %d bytes", len(body))
	}
}

// ── JSON helpers ────────────────────────────────

func TestWriteError_Format(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, 401, "UNAUTHENTICATED", "test msg")
	if w.Code != 401 {
		t.Errorf("status = %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
	var env errorEnvelope
	json.Unmarshal(w.Body.Bytes(), &env)
	if env.Error.Code != "UNAUTHENTICATED" {
		t.Errorf("code = %q", env.Error.Code)
	}
	if env.Error.Message != "test msg" {
		t.Errorf("message = %q", env.Error.Message)
	}
}

func TestWriteJSON_Format(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, 200, map[string]string{"hello": "world"})
	if w.Code != 200 {
		t.Errorf("status = %d", w.Code)
	}
	var m map[string]string
	json.Unmarshal(w.Body.Bytes(), &m)
	if m["hello"] != "world" {
		t.Errorf("hello = %q", m["hello"])
	}
}

// ── Emulator support ────────────────────────────

var emulatorCfg = firebaseConfig{
	ProjectID:        testProjectID,
	APIKey:           "AIzaSyTestKey",
	AuthDomain:       "test-project-123.firebaseapp.com",
	AuthEmulatorHost: "localhost:9099",
}

func signUnsignedToken(t *testing.T, claims firebaseClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	s, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("signing unsigned token: %v", err)
	}
	return s
}

func TestVerifyEmulatorToken_Valid(t *testing.T) {
	tok := signUnsignedToken(t, validClaims())
	u, err := verifyEmulatorToken(tok, testProjectID)
	if err != nil {
		t.Fatalf("verifyEmulatorToken: %v", err)
	}
	if u.UID != "user-uid-abc123" {
		t.Errorf("uid = %q, want user-uid-abc123", u.UID)
	}
	if u.Email != "jane@example.com" {
		t.Errorf("email = %q", u.Email)
	}
	if u.Name != "Jane Doe" {
		t.Errorf("name = %q", u.Name)
	}
}

func TestVerifyEmulatorToken_EmptySubject(t *testing.T) {
	c := validClaims()
	c.Subject = ""
	tok := signUnsignedToken(t, c)
	_, err := verifyEmulatorToken(tok, testProjectID)
	if err == nil {
		t.Error("expected error for empty subject")
	}
}

func TestVerifyEmulatorToken_AcceptsRS256(t *testing.T) {
	kid := "emu-rs256"
	pk := generateTestKey(t, kid)
	tok := signToken(t, pk, kid, validClaims())
	u, err := verifyEmulatorToken(tok, testProjectID)
	if err != nil {
		t.Fatalf("verifyEmulatorToken should accept RS256 too: %v", err)
	}
	if u.UID != "user-uid-abc123" {
		t.Errorf("uid = %q", u.UID)
	}
}

func TestEmulatorMux_ValidUnsignedToken_200(t *testing.T) {
	srv := httptest.NewServer(newMux(emulatorCfg))
	defer srv.Close()
	tok := signUnsignedToken(t, validClaims())
	req, _ := http.NewRequest("GET", srv.URL+"/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/me: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200; body = %s", resp.StatusCode, b)
	}
	var u userClaims
	json.NewDecoder(resp.Body).Decode(&u)
	if u.UID != "user-uid-abc123" {
		t.Errorf("uid = %q", u.UID)
	}
}

func TestEmulatorMux_GarbageToken_401(t *testing.T) {
	srv := httptest.NewServer(newMux(emulatorCfg))
	defer srv.Close()
	req, _ := http.NewRequest("GET", srv.URL+"/api/me", nil)
	req.Header.Set("Authorization", "Bearer garbage")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/me: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestEmulatorConnectSnippet_WhenSet(t *testing.T) {
	snippet := emulatorConnectSnippet(emulatorCfg)
	if !strings.Contains(snippet, "connectAuthEmulator") {
		t.Error("snippet missing connectAuthEmulator call")
	}
	if !strings.Contains(snippet, "localhost:9099") {
		t.Error("snippet missing emulator host")
	}
}

func TestEmulatorConnectSnippet_WhenEmpty(t *testing.T) {
	snippet := emulatorConnectSnippet(testCfg)
	if snippet != "" {
		t.Errorf("snippet should be empty for production config, got %q", snippet)
	}
}

func TestEmulatorHomePage_HasConnectEmulator(t *testing.T) {
	srv := httptest.NewServer(newMux(emulatorCfg))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "connectAuthEmulator") {
		t.Error("emulator home page missing connectAuthEmulator call")
	}
}

func TestProductionHomePage_NoConnectEmulator(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	s := string(body)
	// The import will still be there, but the actual connectAuthEmulator() call should not
	// Check that no connectAuthEmulator(auth, "http://...) call is present
	if strings.Contains(s, `connectAuthEmulator(auth`) {
		t.Error("production home page should not call connectAuthEmulator")
	}
}

// ── HTTP method enforcement ─────────────────────

func TestHome_POST_Rejected(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	resp, err := http.Post(srv.URL+"/", "text/plain", nil)
	if err != nil {
		t.Fatalf("POST /: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 405 && resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404 or 405", resp.StatusCode)
	}
}

func TestAPIMe_POST_Rejected(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	resp, err := http.Post(srv.URL+"/api/me", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/me: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 405 && resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404 or 405", resp.StatusCode)
	}
}
