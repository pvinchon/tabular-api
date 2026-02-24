# Implementation Plan: Firebase Authentication

**Branch**: `002-firebase-auth` | **Date**: 2026-02-23 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/002-firebase-auth/spec.md`

## Summary

Add Google sign-in, sign-out, and profile viewing to the
existing Go web server using Firebase Authentication. The
browser-side flow uses the Firebase JS SDK (CDN, pinned version)
for Google sign-in. The server verifies Firebase ID tokens using
manual JWT verification with `golang-jwt/jwt` against Google's
public keys (no full Admin SDK). An API-First `GET /api/me`
endpoint returns the authenticated user's profile as JSON. A
`/profile` page serves static HTML that handles auth state
client-side.

## Technical Context

**Language/Version**: Go 1.24
**Primary Dependencies**: `github.com/golang-jwt/jwt/v4` v4.5.2 (JWT verification); Firebase JS SDK v11.x via CDN (client-side)
**Storage**: N/A (no server-side persistence; Firebase manages auth state)
**Testing**: `go test` (internal), `./bin/test.sh` (integration)
**Target Platform**: Linux container (Docker)
**Project Type**: web-service
**Performance Goals**: < 2s profile page load (SC-002); < 30s full sign-in flow (SC-001)
**Constraints**: Single new Go dependency; no service account key; injected config only
**Scale/Scope**: 3 new endpoints (`/profile`, `/api/me`, existing `/` unchanged); ~300 LOC added

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| # | Principle | Status | Notes |
|---|-----------|--------|-------|
| I | General Correctness | PASS | Token verification follows Firebase/JWT standards, not test-specific behavior |
| II | Test Isolation | PASS | No reading of integration test source; test info from user context only |
| III | Environment Isolation | PASS | All Firebase config via env vars (`FIREBASE_PROJECT_ID`, `FIREBASE_API_KEY`, `FIREBASE_AUTH_DOMAIN`); fail-fast if missing; no default credentials |
| IV | Data Durability | N/A | No server-side user data storage |
| V | Network Boundaries | PASS | Single outbound call to `googleapis.com` for public keys (documented, cached, required for operation) |
| VI | Build & Runtime Integrity | PASS | Go version pinned; `golang-jwt` pinned to exact version; Firebase JS SDK version pinned in CDN URL; deterministic Docker build |
| VII | Simplicity | PASS | Manual JWT verification (1 dep) instead of full Admin SDK (~50 deps); no template engine; inline HTML strings |
| VIII | API-First Design | PASS | `GET /api/me` returns JSON profile data; error envelope defined; contract documented before implementation |
| IX | Type Safety | PASS | Go structs for user profile, JWT claims, and error responses; typed validation |
| X | Observability | PASS | Existing logging middleware covers all new endpoints; no secrets logged |
| — | Filesystem Boundary | PASS | All code in `implementations/`; specs in `specs/` |
| — | Prohibited Behaviors | PASS | No test skipping, no feature flags, no embedded credentials |

**Pre-design gate**: PASS
**Post-design gate**: PASS

## Project Structure

### Documentation (this feature)

```text
specs/002-firebase-auth/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── http-api.md
├── checklists/
│   └── requirements.md
└── tasks.md             # Phase 2 (/speckit.tasks - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
implementations/
├── Dockerfile
├── go.mod
├── go.sum              # New: dependency checksum file
└── main.go
```

**Structure Decision**: Flat single-file layout, same as 001.
The server grows to ~300-400 LOC with auth handlers, JWT
verification, and HTML templates. This remains manageable in a
single `main.go` file given the project's simplicity principle.
If future features push beyond ~500 LOC, splitting into packages
should be revisited. The `go.sum` file is new — required by the
addition of the `golang-jwt` dependency.

## Complexity Tracking

No constitution violations to justify. The design is minimal:
one new Go dependency, three new HTTP handlers, inline HTML,
client-side Firebase JS via CDN.
