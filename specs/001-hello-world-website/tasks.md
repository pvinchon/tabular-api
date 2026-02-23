# Tasks: Hello World Website

**Input**: Design documents from `specs/001-hello-world-website/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/http-api.md, quickstart.md

**Tests**: Not explicitly requested in the feature specification. No test tasks generated.

**Organization**: Tasks grouped by user story for independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2)
- Include exact file paths in descriptions

## Phase 1: Setup

**Purpose**: Initialize Go project and Docker configuration

- [x] T001 Initialize Go module in implementations/go.mod pinning Go 1.24
- [x] T002 [P] Create multi-stage Dockerfile in implementations/Dockerfile (golang:1.24-alpine build stage, scratch runtime stage)

---

## Phase 2: Foundational

**Purpose**: Server startup, configuration, and logging infrastructure that all user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T003 Implement PORT environment variable reading with fail-fast (log error + exit 1 if missing) in implementations/main.go
- [x] T004 Implement HTTP server startup on configured port with graceful mux setup in implementations/main.go
- [x] T005 Implement structured JSON request logging middleware using log/slog in implementations/main.go (timestamp, request_id, method, path, status, latency_ms per contracts/http-api.md)

**Checkpoint**: Server starts on configured port, logs requests, exits on missing config

---

## Phase 3: User Story 1 — View Hello World Page (Priority: P1) 🎯 MVP

**Goal**: Visitor navigates to `/` and sees "Hello, World!" displayed on the page

**Independent Test**: `curl http://localhost:<port>/` returns HTTP 200 with body containing "Hello, World!"

### Implementation for User Story 1

- [x] T006 [US1] Register GET / handler that responds with HTTP 200 and HTML body containing "Hello, World!" in implementations/main.go
- [x] T007 [US1] Register catch-all handler that responds with HTTP 404 and empty body for all undefined routes in implementations/main.go

**Checkpoint**: `./bin/test.sh` passes — US1 is the MVP

---

## Phase 4: User Story 2 — Valid HTML Document (Priority: P2)

**Goal**: The page at `/` is a well-formed HTML5 document with charset, title, and mobile viewport

**Independent Test**: The HTML served at `/` passes W3C validation with zero errors; page is legible on mobile without zooming

### Implementation for User Story 2

- [x] T008 [US2] Upgrade the GET / response body to a complete HTML5 document (DOCTYPE, html lang, meta charset=utf-8, meta viewport, title, h1) per contracts/http-api.md in implementations/main.go
- [x] T009 [US2] Set Content-Type response header to text/html; charset=utf-8 in implementations/main.go

**Checkpoint**: Full HTML5 page served; both US1 and US2 independently validated

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Final verification and documentation

- [x] T010 Verify Dockerfile builds successfully and produces working container image
- [x] T011 Run quickstart.md validation (local build + Docker build + test script)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on T001 (go.mod exists for compilation)
- **User Story 1 (Phase 3)**: Depends on Phase 2 completion (server running)
- **User Story 2 (Phase 4)**: Depends on T006 (upgrades existing handler)
- **Polish (Phase 5)**: Depends on all previous phases

### User Story Dependencies

- **User Story 1 (P1)**: Depends on Foundational only — no cross-story deps
- **User Story 2 (P2)**: Depends on US1 (refines the same handler) — sequential

### Within Each Phase

- T001 and T002 are parallel (different files)
- T003 before T004 (config needed before server start)
- T004 before T005 (server needed before middleware)
- T006 before T007 (main route before catch-all)
- T008 before T009 (body before headers, logically)

### Parallel Opportunities

- T001 and T002 can run in parallel (go.mod and Dockerfile are independent files)
- All other tasks are sequential due to single-file layout (main.go)

---

## Parallel Example: Setup Phase

```
# These can run simultaneously:
Task T001: "Initialize Go module in implementations/go.mod"
Task T002: "Create multi-stage Dockerfile in implementations/Dockerfile"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001, T002)
2. Complete Phase 2: Foundational (T003, T004, T005)
3. Complete Phase 3: User Story 1 (T006, T007)
4. **STOP and VALIDATE**: Run `./bin/test.sh` — should pass
5. This is the MVP — a working Hello World page in a Docker container

### Incremental Delivery

1. Setup + Foundational → Server runs, logs requests, fails fast on missing config
2. Add User Story 1 → `./bin/test.sh` passes (MVP!)
3. Add User Story 2 → Valid HTML5 document with mobile support
4. Polish → Final verification

---

## Notes

- All implementation is in a single file (implementations/main.go) so most tasks are sequential
- T002 (Dockerfile) is the only task that touches a different file and can be parallelized
- No test tasks generated — tests were not requested in spec
- 11 total tasks across 5 phases
- Task IDs are sequential in execution order (T001–T011)
