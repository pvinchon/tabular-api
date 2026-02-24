````markdown
# HTTP API Contract: Firebase Authentication

**Feature**: 002-firebase-auth
**Date**: 2026-02-23
**Protocol**: HTTP/1.1

## Error Envelope

All API endpoints (paths under `/api/`) return JSON. Error
responses use a uniform envelope:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable description"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `error.code` | string | Machine-readable error code (uppercase, underscore-separated) |
| `error.message` | string | Human-readable error description |

## Endpoints

### `GET /` (unchanged from 001)

Serves the Hello World HTML page. No authentication required.

**Request**: No parameters, no headers required.

**Response (200)**: Same as 001-hello-world-website contract.

**Invariants**:
- Response body MUST contain the text `Hello, World!`
- Response MUST be valid HTML5
- No authentication required

---

### `GET /profile`

Serves the profile page HTML shell. This is a static HTML page
containing client-side JavaScript that handles the Firebase
sign-in flow and profile rendering.

**Request**: No server-side authentication required (auth is
enforced client-side — the page loads, then JS checks auth
state and either renders the profile or initiates sign-in).

**Response (200)**:

```
HTTP/1.1 200 OK
Content-Type: text/html; charset=utf-8
```

**Invariants**:
- Response MUST be valid HTML5
- Response MUST contain the Firebase JS SDK initialization code
- Firebase configuration values MUST be injected from
  server-side environment variables (not hardcoded)

---

### `GET /api/me`

Returns the authenticated user's profile data.

**Request**:

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | Yes | `Bearer <firebase-id-token>` |

**Response (200)** — valid token:

```
HTTP/1.1 200 OK
Content-Type: application/json

{
  "uid": "abc123def456",
  "email": "user@example.com",
  "name": "Jane Doe",
  "picture": "https://lh3.googleusercontent.com/a/photo"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `uid` | string | Firebase user ID |
| `email` | string | User's email address |
| `name` | string | Display name from Google account |
| `picture` | string | Profile photo URL. Empty string `""` if not set. |

**Response (401)** — missing or invalid token:

```
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{
  "error": {
    "code": "UNAUTHENTICATED",
    "message": "Missing or invalid authentication token"
  }
}
```

**Invariants**:
- Response MUST be `application/json`
- A valid, non-expired Firebase ID token MUST be present in the
  `Authorization` header
- The server MUST verify the token signature against Google's
  public keys
- The server MUST verify `aud` matches the configured Firebase
  project ID
- The server MUST verify `iss` matches
  `https://securetoken.google.com/<projectId>`
- The server MUST verify `exp` is in the future and `iat` is in
  the past
- A `401` response MUST use the error envelope format

---

### Any undefined route (unchanged from 001)

**Response (404)**:

```
HTTP/1.1 404 Not Found
Content-Length: 0
```

**Invariants**:
- Response body MUST be empty
- Status code MUST be 404

## Structured Logging

Every request produces a structured JSON log entry (same format
as 001, Constitution Principle X):

```json
{
  "time": "2026-02-23T12:00:00.000Z",
  "level": "INFO",
  "msg": "request",
  "request_id": "abc-123",
  "method": "GET",
  "path": "/api/me",
  "status": 200,
  "latency_ms": 1.2
}
```

For authentication failures, the log entry includes the status
code `401` but MUST NOT log the token value or any user
credentials (Constitution Principle X: no secrets in logs).

## Authentication Flow Summary

```
Browser                          Go Server                Google
  │                                 │                        │
  │  1. signInWithPopup()           │                        │
  │────────────────────────────────────────────────────────▶ │
  │  2. Google OAuth consent        │                        │
  │◀────────────────────────────────────────────────────────│
  │  3. Firebase returns ID token   │                        │
  │                                 │                        │
  │  4. GET /api/me                 │                        │
  │     Authorization: Bearer <jwt> │                        │
  │────────────────────────────────▶│                        │
  │                                 │  5. Fetch public keys  │
  │                                 │     (cached)           │
  │                                 │───────────────────────▶│
  │                                 │◀───────────────────────│
  │                                 │  6. Verify JWT         │
  │  7. 200 OK { user profile }     │                        │
  │◀────────────────────────────────│                        │
```

````
