# Research: Firebase Authentication

**Feature**: 002-firebase-auth
**Date**: 2026-02-23

---

## Topic 1: Firebase Admin SDK for Go (Server-Side Token Verification)

### Findings

**Package**: `firebase.google.com/go/v4` (latest v4.19.0,
published 2026-01-21)

**Import paths**:

```go
import (
    firebase "firebase.google.com/go/v4"
    "firebase.google.com/go/v4/auth"
)
```

**Initialization and verification**:

```go
// Initialize with explicit project ID and service account JSON
opt := option.WithCredentialsFile("/path/to/serviceAccountKey.json")
conf := &firebase.Config{ProjectID: "my-project-id"}
app, err := firebase.NewApp(ctx, conf, opt)

// Get auth client
authClient, err := app.Auth(ctx)

// Verify an ID token
token, err := authClient.VerifyIDToken(ctx, idTokenString)
// token.UID contains the user's Firebase UID
// token.Claims contains additional claims
```

**Credentials/Configuration required**:

- A **service account JSON key file** or the
  `GOOGLE_APPLICATION_CREDENTIALS` environment variable pointing
  to one. The constitution (Principle III) requires credentials
  to be explicitly provided — no auto-resolution.
- The **Firebase project ID** — can be passed in
  `firebase.Config{ProjectID: "..."}` or via
  `GOOGLE_CLOUD_PROJECT` env var, or read from the service
  account JSON.

**`VerifyIDToken` behavior** (from official docs): In
non-emulator mode, this function does **not** make RPC calls
most of the time. The only time it makes an RPC call is when
Google public keys need to be refreshed. These keys get cached
up to 24 hours.

**Returned `auth.Token` struct**:

```go
type Token struct {
    AuthTime int64                  `json:"auth_time"`
    Issuer   string                 `json:"iss"`
    Audience string                 `json:"aud"`
    Expires  int64                  `json:"exp"`
    IssuedAt int64                  `json:"iat"`
    Subject  string                 `json:"sub,omitempty"`
    UID      string                 `json:"uid,omitempty"`
    Firebase FirebaseInfo           `json:"firebase"`
    Claims   map[string]interface{} `json:"-"`
}

type FirebaseInfo struct {
    SignInProvider string                 `json:"sign_in_provider"`
    Tenant        string                 `json:"tenant"`
    Identities    map[string]interface{} `json:"identities"`
}
```

**Dependencies pulled in by the full Admin SDK** (from go.mod):

The SDK's direct `require` block includes:

| Dependency | Purpose |
|---|---|
| `cloud.google.com/go/firestore` | Firestore client |
| `cloud.google.com/go/storage` | Cloud Storage client |
| `github.com/MicahParks/keyfunc` | JWKS key fetching |
| `github.com/golang-jwt/jwt/v4` | JWT parsing/validation |
| `golang.org/x/oauth2` | OAuth2 support |
| `google.golang.org/api` | Google API client core |
| `google.golang.org/appengine/v2` | App Engine compat |

Transitive dependencies include gRPC, protobuf, OpenTelemetry,
cloud monitoring, and many others (~50+ indirect packages). This
is a **heavy** dependency graph for a project that currently has
zero external dependencies.

**Lighter alternative — manual JWT verification**:

Firebase ID tokens are standard RS256 JWTs. Google publishes
the public keys at:
`https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com`

Manual verification requires:

1. Fetch the public keys (X.509 certs) from the URL above.
2. Cache them using the `max-age` from the `Cache-Control`
   response header.
3. Verify the JWT header: `alg` = `RS256`, `kid` matches one of
   the fetched keys.
4. Verify the JWT payload: `exp` in the future, `iat` in the
   past, `aud` = Firebase project ID, `iss` =
   `https://securetoken.google.com/<projectId>`, `sub` is
   non-empty.
5. Verify the signature against the corresponding public key.

This approach requires only:
- A JWT library (e.g. `github.com/golang-jwt/jwt/v4`)
- Standard library `crypto/x509`, `crypto/rsa`, `net/http`
- No service account key needed (only the project ID)

### Decision

**Use manual JWT verification** instead of the full Firebase
Admin SDK.

### Rationale

1. **Constitution VII (Simplicity)**: The Admin SDK pulls in
   Firestore, Cloud Storage, gRPC, protobuf, OpenTelemetry, and
   ~50 transitive dependencies. The project only needs token
   verification — one function.
2. **Constitution VI (Build & Runtime Integrity)**: Fewer
   dependencies means a smaller attack surface and simpler
   pinning.
3. **Constitution III (Environment Isolation)**: Manual
   verification needs only a project ID string (injected via
   env var), not a service account JSON key. This simplifies
   credential management.
4. **Dependency count**: Manual approach adds 1 direct dependency
   (`github.com/golang-jwt/jwt/v4`) vs. the Admin SDK's ~7
   direct + ~50 indirect.
5. **No service account key required**: `VerifyIDToken` only
   fetches Google's public keys (no privileged access needed).
   The Admin SDK requires a service account key even though
   verification doesn't use it for signing — only for SDK
   initialization.

### Alternatives Considered

| Alternative | Pros | Cons |
|---|---|---|
| Firebase Admin SDK (`firebase.google.com/go/v4`) | Official, battle-tested, handles key rotation automatically | Heavy dependency tree (~50+ packages), requires service account JSON, pulls in Firestore/Storage/gRPC |
| Manual JWT with `golang-jwt` + stdlib | Minimal deps (1 package), no service account key needed, full control | Must implement key caching and rotation ourselves |
| Manual JWT with stdlib only (no JWT library) | Zero external dependencies | Significant effort to implement RS256 verification correctly; error-prone |

---

## Topic 2: Client-Side Firebase Auth Flow

### Findings

**Browser-side flow with Firebase JS SDK (Google sign-in)**:

1. Load Firebase JS SDK via CDN `<script>` tags (no npm/bundler
   required).
2. Initialize Firebase with a client-side config object
   (`apiKey`, `authDomain`, `projectId`).
3. Create a `GoogleAuthProvider` instance.
4. Call `signInWithPopup(auth, provider)` or
   `signInWithRedirect(auth, provider)`.
5. Google's sign-in UI appears (popup or redirect).
6. On success, Firebase returns a `UserCredential` object.

**Getting the ID token to send to the server**:

```javascript
import { getAuth } from "firebase/auth";

const auth = getAuth();
const user = auth.currentUser;
const idToken = await user.getIdToken(/* forceRefresh */ false);
```

`getIdToken()` returns a JWT string. It automatically refreshes
the token if it's expired or about to expire. The token is valid
for 1 hour.

**CDN usage (no npm/bundler)**:

Yes, Firebase provides CDN-hosted ES module builds. As of
Firebase JS SDK v9+, the modular API can be loaded via:

```html
<script type="module">
  import { initializeApp } from "https://www.gstatic.com/firebasejs/11.x.x/firebase-app.js";
  import { getAuth, signInWithPopup, GoogleAuthProvider, onAuthStateChanged, signOut }
    from "https://www.gstatic.com/firebasejs/11.x.x/firebase-auth.js";
</script>
```

This requires no npm, no bundler, and no build step. Versions
must be pinned in the URL (Constitution VI: pinned
dependencies).

**Auth state persistence**: Firebase Auth in the browser
persists the session in IndexedDB by default. This means
refreshing the page does not lose the auth state (satisfies
FR-008). `onAuthStateChanged(auth, callback)` fires when auth
state changes (sign-in, sign-out, token refresh).

### Decision

**Use Firebase JS SDK via CDN `<script type="module">` tags**
with the modular v9+ API for client-side auth.

### Rationale

1. No build tooling needed — aligns with the project's
   simplicity (stdlib Go server, no frontend build pipeline).
2. Version can be pinned in the CDN URL (Constitution VI).
3. The modular API is tree-shakeable conceptually (we only
   import what we use).
4. Firebase handles all the OAuth complexity client-side.

### Alternatives Considered

| Alternative | Pros | Cons |
|---|---|---|
| Firebase JS SDK via CDN (chosen) | No build step, pinned versions, officially supported | Must use `type="module"` in script tag |
| Firebase JS SDK via npm + bundler | Tree-shaking, TypeScript support | Requires Node.js toolchain, webpack/vite/esbuild — unnecessary complexity |
| Google Sign-In API directly (no Firebase) | Fewer dependencies | Must handle token exchange manually, Firebase session management lost |

---

## Topic 3: Docker Implications

### Findings

**Current image**: `FROM scratch` — a completely empty base
image. Contains only the statically compiled Go binary.

**The problem**: The Firebase Admin SDK (and manual JWT
verification) make HTTPS calls to
`https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com`
to fetch public keys for token verification. HTTPS requires TLS
certificate verification, which needs CA (Certificate Authority)
root certificates in the container. A `scratch` image has **no
CA certificates**, so HTTPS calls will fail with
`x509: certificate signed by unknown authority`.

**Options for CA certificates**:

| Base Image | Size | Has CA Certs | Notes |
|---|---|---|---|
| `FROM scratch` | 0 B | No | Must manually copy certs |
| `gcr.io/distroless/static` | ~2 MB | Yes | Google-maintained, no shell, no package manager, includes CA certs and timezone data |
| `alpine` | ~8 MB | Yes | Has shell and `apk` package manager |
| `scratch` + manual cert copy | 0 B + ~200 KB | Yes | `COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/` |

**Manual cert copy approach** (keeps scratch base):

```dockerfile
FROM golang:1.24-alpine AS build
# ... build steps ...

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /app/server /server
ENTRYPOINT ["/server"]
```

This copies the CA certificate bundle from the build stage into
the scratch image, adding ~200 KB.

### Decision

**Keep `FROM scratch` and copy CA certificates from the build
stage.**

### Rationale

1. **Constitution VII (Simplicity)**: Minimal change — one
   additional `COPY` line in the Dockerfile.
2. **Smallest image**: `scratch` + certs (~200 KB overhead) is
   smaller than `distroless/static` (~2 MB).
3. **No additional dependencies**: No new base image to track or
   pin.
4. **Deterministic**: The certs come from the pinned
   `golang:1.24-alpine` build image (Constitution VI).
5. **Already proven**: The Go Alpine build image already contains
   the certificates; we just need to copy them.

### Alternatives Considered

| Alternative | Pros | Cons |
|---|---|---|
| `scratch` + copied certs (chosen) | Smallest, no new deps, one-line change | Certs tied to build image version |
| `gcr.io/distroless/static` | Official, maintained, includes certs + tzdata | Adds ~2 MB, introduces new base image dependency to track |
| `alpine` | Has shell for debugging, package manager | ~8 MB, includes unnecessary tools, larger attack surface |

---

## Topic 4: Architecture Pattern for Server-Verified Auth

### Findings

**Standard pattern** (client-side Firebase + server-side
verification):

```
┌──────────┐    ┌─────────────┐    ┌───────────────────┐
│  Browser  │───▶│  Go Server  │───▶│ Google Public Keys │
│           │    │             │    │ (googleapis.com)   │
│ Firebase  │    │ Verify JWT  │    └───────────────────┘
│ JS SDK    │    │ Return JSON │
└──────────┘    └─────────────┘
```

1. **Client** signs in via Firebase JS SDK → gets ID token.
2. **Client** sends ID token as `Authorization: Bearer <token>`
   header on requests to Go server.
3. **Server** extracts token from header, verifies signature and
   claims against Google's public keys, extracts user info
   (UID, email, display name, photo URL) from the token claims.
4. **Server** returns JSON response with user data.

**Profile page architecture — API-First (Constitution VIII)**:

The spec says "the server MUST validate the Firebase ID token
before serving protected pages." Combined with Constitution
VIII (API-First), this means:

- The server exposes a **JSON API endpoint** (e.g.
  `GET /api/me`) that returns the authenticated user's profile
  data as JSON.
- The server serves a **static HTML page** for the profile page
  URL. This page contains client-side JavaScript that:
  1. Gets the current user's ID token from Firebase Auth.
  2. Calls the API endpoint with the Bearer token.
  3. Renders the profile data in the page.
- If the user is not authenticated, the client-side JS initiates
  the Google sign-in flow (FR-009).

**Why this pattern (not server-rendered HTML)**:

- **Constitution VIII**: "Every feature MUST be designed as an
  API endpoint before any client or UI work begins." The
  profile data retrieval must be an API endpoint returning JSON.
- **Constitution VIII**: "All endpoints MUST return consistent
  JSON responses with a uniform error envelope."
- Server-rendered HTML (where the server embeds user data into
  HTML templates) would bypass the API layer — the profile data
  would only be available as rendered HTML, not as a structured
  API response.

**Token claims available without server-side user lookup**:

Firebase ID tokens include standard claims (`sub`/`uid`,
`email`, `name`, `picture`) when the user signed in with Google.
However, these are in the token's custom claims, not in the
standard JWT fields. The `name` and `picture` fields are
available in the token's `Claims` map — **no additional API
call to Firebase is needed** to get basic profile data.

Specifically, for a Google sign-in, the ID token typically
contains:

| Claim | Description |
|---|---|
| `sub` / `uid` | Firebase user ID |
| `email` | User's email address |
| `email_verified` | Whether email is verified |
| `name` | Display name |
| `picture` | Profile photo URL |
| `firebase.sign_in_provider` | e.g. `google.com` |

**Request/response pattern**:

```
GET /api/me
Authorization: Bearer <firebase-id-token>

200 OK
Content-Type: application/json

{
  "uid": "abc123",
  "email": "user@example.com",
  "name": "Jane Doe",
  "picture": "https://lh3.googleusercontent.com/..."
}
```

```
GET /api/me
(no Authorization header or invalid token)

401 Unauthorized
Content-Type: application/json

{
  "error": {
    "code": "UNAUTHENTICATED",
    "message": "Missing or invalid authentication token"
  }
}
```

### Decision

**API-First architecture**: Server exposes JSON API endpoints;
client-side JavaScript handles auth flow and rendering.

### Rationale

1. **Constitution VIII (API-First)**: Every feature must be an
   API endpoint first. Profile data flows through a JSON API.
2. **Constitution VIII**: Consistent JSON responses with uniform
   error envelope.
3. **Separation of concerns**: Server handles authentication
   verification and data; client handles rendering and auth
   flow.
4. **Testability**: API endpoints are independently testable
   (curl, integration tests) without a browser.
5. **No server-side template engine needed** — the server
   continues to serve static HTML strings with embedded
   JavaScript, same pattern as the existing Hello World page.

### Alternatives Considered

| Alternative | Pros | Cons |
|---|---|---|
| API-First (JSON endpoints + client rendering) (chosen) | Constitution-compliant, testable, clean separation | Slightly more client-side JS |
| Server-rendered HTML (Go templates) | Simpler client JS, SSR | Violates Constitution VIII (data not available as API), introduces template engine dependency |
| SPA with separate static file server | Full separation of frontend/backend | Overcomplicated for the current scope, violates Constitution VII |

---

## Summary of Key Decisions

| # | Topic | Decision |
|---|---|---|
| 1 | Server-side token verification | Manual JWT verification with `golang-jwt/jwt` + stdlib (not full Admin SDK) |
| 2 | Client-side auth | Firebase JS SDK v9+ via CDN `<script type="module">`, pinned version |
| 3 | Docker base image | Keep `FROM scratch`, copy CA certs from build stage |
| 4 | Architecture pattern | API-First: JSON endpoints + client-side rendering |

## Configuration Requirements (from research)

| Variable | Required | Description |
|---|---|---|
| `PORT` | Yes | TCP port (existing) |
| `FIREBASE_PROJECT_ID` | Yes | Firebase project ID for token verification |
| `FIREBASE_API_KEY` | Yes | Client-side Firebase config (injected into HTML) |
| `FIREBASE_AUTH_DOMAIN` | Yes | Client-side Firebase config (injected into HTML) |

Note: No service account JSON key is required. Manual JWT
verification only needs the project ID. The public keys are
fetched from a well-known Google URL.

## Outbound Network Calls (Constitution V compliance)

| Endpoint | Purpose | Required |
|---|---|---|
| `https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com` | Fetch public keys for JWT signature verification | Yes (cached, refreshed per `Cache-Control` max-age) |

This is the only outbound call. It is to an explicitly known
Google endpoint required for the feature's core operation
(token verification). It must be documented as a configured
dependency.

## New Dependencies

| Dependency | Version | Purpose |
|---|---|---|
| `github.com/golang-jwt/jwt/v4` | v4.5.2 (pin exact) | JWT parsing and RS256 signature verification |

This is the only new Go dependency. The Firebase Admin SDK's
`go.mod` also uses this same library internally.
