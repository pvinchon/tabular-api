# Tasks: Firebase Authentication

**Input**: Design documents from `specs/002-firebase-auth/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/http-api.md, quickstart.md

**Tests**: Not explicitly requested in the feature specification. No test tasks generated.

**Organization**: Tasks grouped by user story for independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup

**Purpose**: Add the new Go dependency and update Docker/Compose configuration

- [x] T001 Add `github.com/golang-jwt/jwt/v4` v4.5.2 dependency to implementations/go.mod and generate implementations/go.sum
- [x] T002 [P] Update implementations/Dockerfile to copy CA certificates from build stage (`COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/`)
- [x] T003 [P] Add `FIREBASE_PROJECT_ID`, `FIREBASE_API_KEY`, and `FIREBASE_AUTH_DOMAIN` environment variables to compose.yaml

---

## Phase 2: Foundational

**Purpose**: Core auth infrastructure that ALL user stories depend on — config validation, JWT verification, error envelope

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T004 Add fail-fast validation for `FIREBASE_PROJECT_ID`, `FIREBASE_API_KEY`, and `FIREBASE_AUTH_DOMAIN` environment variables at startup in implementations/main.go (log error + exit 1 if any missing, per FR-010 and Constitution III)
- [x] T005 Implement Google public key fetcher: fetch X.509 certificates from `https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com`, parse to RSA public keys, cache with expiry from `Cache-Control` `max-age` header, in implementations/main.go
- [x] T006 Implement Firebase ID token verification function: extract Bearer token from Authorization header, parse JWT with `golang-jwt/jwt/v4`, verify `alg`=RS256, `kid` matches cached key, `aud`=project ID, `iss`=`https://securetoken.google.com/<projectId>`, `exp` in future, `iat` in past, `sub` non-empty, return user claims (uid, email, name, picture) in implementations/main.go
- [x] T007 Implement JSON error envelope response helper matching contracts/http-api.md format (`{"error":{"code":"...","message":"..."}}`) in implementations/main.go

**Checkpoint**: Server starts with all config validated; JWT verification and error envelope are ready for use by all user stories

---

## Phase 3: User Story 1 — Sign In with Google (Priority: P1) 🎯 MVP

**Goal**: Visitor clicks "Sign in with Google", completes Google sign-in, and sees their display name on the page

**Independent Test**: Visit `/`, click "Sign in with Google", complete the Google flow, confirm display name appears on the page

### Implementation for User Story 1

- [x] T008 [US1] Update the GET / handler to inject Firebase config (`apiKey`, `authDomain`, `projectId`) from environment variables into the HTML page and add Firebase JS SDK v11.x CDN `<script type="module">` imports (firebase-app.js, firebase-auth.js) with pinned version in implementations/main.go
- [x] T009 [US1] Add client-side JavaScript to the root page: initialize Firebase app, use `onAuthStateChanged` to detect auth state, show "Sign in with Google" button when unauthenticated, show display name + "Sign out" button when authenticated, handle `signInWithPopup` with `GoogleAuthProvider`, handle cancelled/failed sign-in gracefully (FR-001, FR-002, FR-003, FR-008) in implementations/main.go
- [x] T010 [US1] Implement `GET /api/me` handler: extract Bearer token from Authorization header, call token verification function (T006), return JSON response with `uid`, `email`, `name`, `picture` fields on success (200), or error envelope with code `UNAUTHENTICATED` on failure (401), per contracts/http-api.md in implementations/main.go
- [x] T011 [US1] Ensure the existing `GET /` route with "Hello, World!" text remains visible and accessible regardless of authentication state (FR-011) in implementations/main.go

**Checkpoint**: Sign-in works end-to-end; `GET /api/me` returns profile JSON; Hello World text still visible; `./bin/test.sh` still passes

---

## Phase 4: User Story 2 — View Profile (Priority: P2)

**Goal**: Authenticated user navigates to `/profile` and sees their display name, email, and profile picture

**Independent Test**: After signing in, navigate to `/profile`, confirm display name, email, and profile picture are shown; if no picture, a placeholder is shown

### Implementation for User Story 2

- [x] T012 [US2] Register `GET /profile` handler that serves an HTML page with Firebase JS SDK initialization (same config injection as T008), client-side JS that checks auth state on load: if authenticated, calls `GET /api/me` with Bearer token and renders profile (name, email, picture); if no picture field or empty, shows a default placeholder image (FR-004, FR-005); if unauthenticated, automatically initiates `signInWithPopup` (FR-009) in implementations/main.go
- [x] T013 [US2] Ensure the `/profile` route is registered before the catch-all 404 handler so it is served correctly in implementations/main.go

**Checkpoint**: Profile page works independently; unauthenticated access triggers sign-in; placeholder image shown when no picture; US1 still works

---

## Phase 5: User Story 3 — Sign Out (Priority: P3)

**Goal**: Authenticated user clicks "Sign out" and the page returns to the unauthenticated state

**Independent Test**: After signing in, click "Sign out", confirm the page shows the sign-in button and the profile page redirects to sign-in

### Implementation for User Story 3

- [x] T014 [US3] Add sign-out functionality to the root page: "Sign out" button calls Firebase `signOut(auth)`, `onAuthStateChanged` callback switches UI back to unauthenticated state showing "Sign in with Google" button (FR-006, FR-007) in implementations/main.go
- [x] T015 [US3] Add sign-out functionality to the profile page: "Sign out" button calls Firebase `signOut(auth)`, then automatically re-initiates Google sign-in flow (since profile page requires auth per FR-009) in implementations/main.go

**Checkpoint**: Sign-out works on both pages; all 3 user stories work independently; `./bin/test.sh` still passes

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final verification and edge case handling

- [x] T016 Verify structured JSON logging covers all new endpoints (`/profile`, `/api/me`) with correct fields (request_id, method, path, status, latency_ms) and no token values logged (Constitution X) in implementations/main.go
- [x] T017 Verify Dockerfile builds successfully and produces working container image with CA certificates present
- [x] T018 Run quickstart.md validation (local build + Docker build + docker compose up)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on T001 (go.mod has jwt dependency for compilation)
- **User Story 1 (Phase 3)**: Depends on Phase 2 completion (JWT verification + error envelope ready)
- **User Story 2 (Phase 4)**: Depends on T010 (needs `GET /api/me` to fetch profile data)
- **User Story 3 (Phase 5)**: Depends on T009 (sign-out modifies the same auth UI created in sign-in)
- **Polish (Phase 6)**: Depends on all previous phases

### User Story Dependencies

- **User Story 1 (P1)**: Depends on Foundational only — no cross-story deps
- **User Story 2 (P2)**: Depends on US1's `GET /api/me` endpoint (T010) — sequential after US1
- **User Story 3 (P3)**: Depends on US1's auth UI (T009) — sequential after US1; independent of US2

### Within Each Phase

- T001, T002, T003 can run in parallel (different files)
- T004 → T005 → T006 (config before key fetch before verification)
- T007 parallel with T005/T006 (different concern, same file but independent code)
- T008 → T009 (Firebase SDK loaded before auth logic)
- T010 after T009 (API endpoint uses verification from T006, but implemented after client-side code for logical flow)
- T012 → T013 (handler before route ordering)
- T014 and T015 are parallel in principle but both touch main.go

### Parallel Opportunities

- T001, T002, T003 can all run in parallel (go.mod, Dockerfile, compose.yaml)
- Most other tasks are sequential due to single-file layout (implementations/main.go)
- US2 and US3 could run in parallel after US1 (both depend on US1 but not on each other)

---

## Parallel Example: Setup Phase

```
# These can run simultaneously:
Task T001: "Add golang-jwt/jwt/v4 dependency to implementations/go.mod"
Task T002: "Update Dockerfile to copy CA certificates"
Task T003: "Add Firebase env vars to compose.yaml"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001, T002, T003)
2. Complete Phase 2: Foundational (T004, T005, T006, T007)
3. Complete Phase 3: User Story 1 (T008, T009, T010, T011)
4. **STOP and VALIDATE**: Visit `/`, sign in with Google, confirm name shown, run `./bin/test.sh`
5. This is the MVP — sign-in works, Hello World preserved, API endpoint returns user JSON

### Incremental Delivery

1. Setup + Foundational → Server starts with all config, JWT verification ready
2. Add User Story 1 → Sign-in works, `GET /api/me` returns profile JSON (MVP!)
3. Add User Story 2 → Dedicated profile page at `/profile` with full user info
4. Add User Story 3 → Sign-out on both pages
5. Polish → Logging verified, Docker validated, quickstart confirmed

---

## Notes

- All implementation is in a single file (implementations/main.go) so most tasks are sequential
- T002 (Dockerfile) and T003 (compose.yaml) are the only tasks touching different files
- No test tasks generated — tests were not requested in spec
- 18 total tasks across 6 phases
- Task IDs are sequential in execution order (T001–T018)
- Firebase JS SDK version must be pinned in CDN URL (Constitution VI)
- Token values must never appear in logs (Constitution X)
- Sign-in cancelled/declined must not produce an error (US1 acceptance scenario 2)
