# Research: Hello World Website

**Feature**: 001-hello-world-website
**Date**: 2026-02-23

## Technology Selection

### Decision: Go (stdlib only)

**Rationale**:
- Go's `net/http` package provides a production-grade HTTP
  server with zero external dependencies.
- Compiles to a single static binary — no runtime needed in the
  container (can use `scratch` or `distroless` base image).
- Produces Docker images under 10 MB.
- Aligns with Constitution Principle VII (Simplicity): no
  package manager, no dependency resolution, no lock files.
- Aligns with Constitution Principle VI (Build & Runtime
  Integrity): reproducible builds with pinned Go version,
  deterministic binary output.
- Type-safe by default (Constitution Principle IX).
- Fast startup time (< 100ms) satisfies SC-001.

**Alternatives considered**:

| Alternative | Why rejected |
|-------------|-------------|
| Node.js | Requires Node runtime in container (~150 MB image). `http` module is sufficient but adds unnecessary image size. |
| Python | Requires Python runtime. `http.server` is not production-grade (single-threaded, no graceful shutdown). |
| Rust | Excellent fit technically but higher compile times and more ceremony for a trivial feature. Violates Simplicity for this scope. |
| Static file + nginx | Would satisfy FR-001/FR-002 but cannot implement FR-005 (port from env) or FR-003 (custom 404 behavior) without nginx config templating, adding complexity. |

## Port Configuration (FR-005)

### Decision: Read `PORT` environment variable

**Rationale**:
- The integration test injects `PORT` as an environment variable.
- Go's `os.Getenv("PORT")` reads it directly.
- If empty, log error to stderr and `os.Exit(1)`.
- No config file parsing, no flags — just one env var.

**Alternatives considered**:
- CLI flag parsing: unnecessary complexity for a single value.
- Config file: violates Simplicity; overkill for one setting.

## Error Handling for Undefined Routes (FR-003)

### Decision: Custom `http.ServeMux` with explicit route registration

**Rationale**:
- Go 1.22+ `http.ServeMux` supports exact path matching with
  `GET /` pattern syntax.
- Register only `/` — all other paths get a custom 404 handler
  that returns status 404 with empty body.
- Default `http.NotFoundHandler` adds a body ("404 page not
  found\n") which violates the clarified FR-003.

**Alternatives considered**:
- Default mux: adds unwanted body to 404 responses.
- Third-party router (chi, gorilla): violates Simplicity; stdlib
  is sufficient.

## Docker Strategy

### Decision: Multi-stage build, `scratch` base image

**Rationale**:
- Stage 1: `golang:1.24-alpine` — compile static binary with
  `CGO_ENABLED=0`.
- Stage 2: `scratch` — copy binary only, no OS, no shell.
- Produces deterministic, minimal image (< 10 MB).
- Aligns with Constitution Principle VI (deterministic Docker
  image, reproducible builds).

**Alternatives considered**:
- Single-stage alpine: works but image is ~300 MB with Go
  toolchain included.
- Distroless: slightly larger than scratch, adds CA certs and
  tzdata we don't need for this feature.

## HTML Document (FR-002)

### Decision: Inline HTML string in Go source

**Rationale**:
- The HTML is static and trivial (< 20 lines).
- Embedding it as a Go string constant avoids filesystem
  dependencies and `embed` directives.
- Includes: `<!DOCTYPE html>`, `<html lang="en">`,
  `<meta charset="utf-8">`, `<meta name="viewport" ...>`,
  `<title>`, `<body>` with "Hello, World!".

**Alternatives considered**:
- `embed.FS`: appropriate for many files but overkill for one
  static page.
- Template rendering: no dynamic content, so unnecessary.

## Observability (Constitution Principle X)

### Decision: Structured logging middleware

**Rationale**:
- Constitution requires structured log entries for every request
  (timestamp, request ID, method, path, status code, latency).
- Implement as HTTP middleware wrapping the mux.
- Use `log/slog` (Go stdlib since 1.21) for structured JSON
  logging — no external dependency.
- Health and readiness endpoints deferred to a future feature.

**Alternatives considered**:
- Third-party logging (zerolog, zap): violates Simplicity; slog
  is sufficient.
- Skip observability: violates Constitution Principle X.
