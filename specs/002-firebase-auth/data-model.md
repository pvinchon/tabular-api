````markdown
# Data Model: Firebase Authentication

**Feature**: 002-firebase-auth
**Date**: 2026-02-23

## Overview

This feature introduces a single domain entity (User) derived
entirely from Firebase ID token claims. No server-side
persistence is required — the authentication service (Firebase)
is the authoritative data store.

## Entities

### User (derived from token claims)

Represents an authenticated person. Not stored server-side.
Extracted from the verified Firebase ID token on each request.

| Field | Type | Source | Required | Description |
|-------|------|--------|----------|-------------|
| `uid` | string | JWT `sub` claim | Yes | Firebase user ID (unique identifier) |
| `email` | string | JWT `email` claim | Yes | User's email address |
| `name` | string | JWT `name` claim | Yes | Display name from Google account |
| `picture` | string | JWT `picture` claim | No | Profile photo URL from Google account. May be absent if the Google account has no photo set. |

**Validation rules**:
- `uid` must be non-empty (verified during token validation).
- `email` must be present (Google sign-in always provides one).
- `name` must be present (Google sign-in always provides one).
- `picture` may be empty — client renders a default placeholder
  when absent (FR-005).

**Relationships**: None. No other entities exist.

**State transitions**: None. The User entity is stateless — it
is derived fresh from the token on each authenticated request.

### Session (not explicitly modeled)

Authentication state is managed entirely by the Firebase JS SDK
in the browser (IndexedDB). The server does not track sessions.
Each request is independently authenticated via the Bearer token.

## Configuration

| Name | Source | Required | Description |
|------|--------|----------|-------------|
| `PORT` | Environment variable | Yes | TCP port the server listens on (existing) |
| `FIREBASE_PROJECT_ID` | Environment variable | Yes | Firebase project ID for server-side JWT verification (`aud` and `iss` claim validation) |
| `FIREBASE_API_KEY` | Environment variable | Yes | Firebase client-side API key (injected into served HTML/JS) |
| `FIREBASE_AUTH_DOMAIN` | Environment variable | Yes | Firebase auth domain for client-side SDK (injected into served HTML/JS) |

All configuration must be injected via environment variables.
Missing values cause error log + exit code 1 (Constitution
Principle III).

## Public Key Cache (internal, not persisted)

The server caches Google's public signing keys in memory to
avoid fetching them on every request.

| Field | Type | Description |
|-------|------|-------------|
| Keys | map of key ID → public key | RSA public keys parsed from X.509 certificates |
| Expiry | timestamp | Derived from `max-age` in the `Cache-Control` response header |

Keys are refreshed when the cache expires. This is the only
outbound network call the server makes.

## Request/Response Structures

### GET /api/me (authenticated)

**Request**: `Authorization: Bearer <firebase-id-token>`

| Direction | Field | Type | Description |
|-----------|-------|------|-------------|
| Response | `uid` | string | Firebase user ID |
| Response | `email` | string | Email address |
| Response | `name` | string | Display name |
| Response | `picture` | string | Profile photo URL (empty string if absent) |

### GET /api/me (unauthenticated / invalid token)

| Direction | Field | Type | Description |
|-----------|-------|------|-------------|
| Response | `error.code` | string | Machine-readable error code |
| Response | `error.message` | string | Human-readable error message |

````
