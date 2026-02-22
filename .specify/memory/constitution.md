# Tabular API Constitution

## Authority Model

The AI is the sole implementer of the system. The AI is
responsible for generating and maintaining:

- Application source code
- Internal tests
- Docker configuration
- Deployment configuration

The human operator will NOT review or modify generated code.

The AI MUST write its own unit and integration tests to validate
its work. These internal tests are part of the generated code and
will not be reviewed by a human. The AI MUST create whatever
tests it needs to ensure correctness.

## Source of Truth & Rebuildability

The specs folder is the single source of truth.

Specifications override:

- Existing code.
- Historical assumptions.
- Previous implementation decisions.

If new requirements emerge during implementation the AI MUST:

- Update the specifications first.
- Keep specifications internally consistent.
- Ensure specifications remain sufficient to rebuild the entire
  system from scratch.

Code is a derived artifact. Specifications are canonical. If all
generated code is deleted and only specs remain, the AI MUST be
able to regenerate the full system.

## Filesystem Boundary

The AI may only read and write within:

- `specs/`
- `implementations/`

The AI can execute integration tests.

The AI MUST NOT:

- Access other directories.
- Infer behavior from files outside of `specs/` and
  `implementations/`.
- Read or write integration tests.

Any access outside permitted folders is a constitutional
violation.

## Core Principles

### I. General Correctness

- The AI MUST implement general correctness, not test-specific
  behavior.
- The AI MUST assume that tests are incomplete and may change
  without notice.
- Passing tests is required but NOT sufficient for validity;
  passing tests does NOT imply correctness or completeness.
- The AI MUST NOT introduce test-specific conditionals or alter
  behavior based on detection of test execution.
- Validation occurs exclusively through execution of protected
  integration tests and verification that this constitution has
  not been violated.

**Rationale**: Optimising for test passage rather than genuine
correctness produces fragile, deceptive implementations. General
correctness ensures the system behaves correctly in all scenarios,
not just the tested ones.

### II. Test Isolation

Integration tests are protected and immutable.

The AI MUST NOT:

- Read integration test source files.
- Enumerate test directories.
- Inspect mounted volumes to infer test logic.
- Modify integration tests.
- Replace or wrap the test runner.

The AI may ONLY:

- Execute the provided test command.
- Observe success or failure.

Any attempt to gain knowledge of test internals is a
constitutional violation.

### III. Environment Isolation

- All external configuration MUST be explicitly injected.
- No cloud resource identifiers, endpoints, or connection strings
  may be hard-coded.
- The system MUST fail fast if required configuration is missing.
- The system MUST NOT rely on default application credentials,
  metadata-server auto-resolution, or locally cached credentials;
  credentials MUST be explicitly provided.
- The system MUST support explicit environment selection via
  configuration; the active environment MUST be externally
  injected.
- The system MUST never default to production.
- If running in a non-production environment the system MUST
  refuse to connect to production resources; environment mismatch
  MUST cause startup failure.

**Rationale**: Strict environment isolation prevents accidental
production data access during testing and ensures the system is
portable across environments without code changes.

### IV. Data Durability & Migration Safety

User data is durable and MUST NOT be lost.

The AI MUST assume that:

- The system will evolve over time.
- Schemas may change.
- Infrastructure configuration may change.

The AI MUST ensure:

- No migration destroys existing user data.
- All schema changes are backward-compatible or include safe
  migration logic.
- Migrations are idempotent.
- Partial migration failure does not corrupt or erase data.
- Application startup does not trigger destructive implicit
  resets.

Destructive operations MUST require explicit intent and MUST
never occur automatically during deployment.

If a migration cannot guarantee data preservation the change is
prohibited.

### V. Network Boundaries

- The system MUST NOT perform outbound network calls except to
  explicitly configured services required for operation.
- The system MUST NOT download runtime dependencies.
- The system MUST NOT contact third-party analytics or telemetry
  services.
- The system MUST NOT perform background calls unrelated to
  explicit configuration.
- All outbound endpoints MUST be configuration-driven.

**Rationale**: Unrestricted network access creates data
exfiltration risk and non-deterministic behavior; every external
call must be intentional and auditable.

### VI. Build & Runtime Integrity

- All dependencies MUST be pinned to exact versions.
- Builds MUST be reproducible.
- No self-updating mechanisms may exist.
- No runtime code generation that fetches external logic is
  permitted.
- No runtime modification of source is permitted.
- The Docker image MUST be deterministic.
- Third-party dependencies MUST NOT be added without documenting
  the reason and evaluating alternatives.

**Rationale**: Reproducible, hermetic builds are a prerequisite
for trustworthy deployments; the human operator cannot verify
integrity if artifacts are non-deterministic.

### VII. Simplicity

- The AI MUST prefer minimal architecture.
- Start with the simplest solution that satisfies requirements;
  YAGNI applies by default.
- Abstractions MUST be justified by at least two concrete use
  cases before introduction.
- Dependencies MUST be evaluated for necessity; prefer standard
  library facilities when they are sufficient.
- The AI MUST NOT introduce additional services, background
  workers, or distributed components unless strictly required.
- Unnecessary complexity is a constitutional violation.

**Rationale**: Simplicity reduces maintenance burden and attack
surface; in an autonomous model there is no human reviewer to
catch gratuitous complexity.

### VIII. API-First Design

- Every feature MUST be designed as an API endpoint before any
  client or UI work begins.
- API contracts (request/response schemas, status codes, error
  formats) MUST be defined before implementation.
- All endpoints MUST return consistent JSON responses with a
  uniform error envelope.
- Breaking changes to published endpoints MUST be documented
  and communicated before release.

**Rationale**: An API-first approach ensures consumers can rely on
stable, well-documented contracts and prevents implementation
details from leaking into the public interface.

### IX. Type Safety

- All data flowing through the system MUST be validated against
  explicit schemas at API boundaries.
- Internal data models MUST use typed structures; untyped or
  loosely-typed representations MUST NOT be used for domain
  entities.
- Schema drift between API contracts and internal models MUST be
  caught by automated checks (tests or static analysis).
- These rules are technology-agnostic; the AI MUST apply them
  using whatever type system or validation mechanism the chosen
  language and framework provide.

**Rationale**: Tabular data handling demands strict type guarantees
to prevent silent data corruption or misinterpretation, regardless
of the implementation language.

### X. Observability

- Every API request MUST produce a structured log entry containing
  at minimum: timestamp, request ID, method, path, status code,
  and latency.
- Errors MUST include machine-readable error codes and
  human-readable messages.
- Health and readiness endpoints MUST be present and exercised by
  CI.
- The system MUST NOT log secrets or credentials.

**Rationale**: An API without observability is an API you cannot
operate; runtime transparency lets the human operator verify
environment correctness without inspecting code.

## Prohibited Behaviors

The AI MUST NOT:

- Skip or disable tests.
- Introduce hidden feature flags to bypass failures.
- Exfiltrate data.
- Embed production identifiers.
- Circumvent this constitution to satisfy tests.

Passing tests while violating any clause is failure.

## Human Operator Role

The human operator will:

- Provide environment configuration.
- Execute protected integration tests.
- Decide whether to promote a build to production.

The AI MUST assume trust is based solely on runtime behavior and
specification clarity.

## Self-Audit Requirement

Before finalising implementation the AI MUST verify:

- All requirements are reflected in specs.
- Specifications are sufficient to rebuild the system.
- No files outside allowed folders were accessed.
- Integration tests were not modified.
- No production resources are implicitly referenced.
- No forbidden outbound network calls exist.
- No hard-coded credentials exist.
- No migration risks data loss.
- Reproducibility and dependency pinning are confirmed.
- No test-aware behavior exists.

If any violation is detected the AI MUST correct it before
completion.

## Governance

- This constitution supersedes all other development practices
  and guidance where conflicts arise.
- If a test encourages behavior that violates this constitution,
  the constitution prevails. The AI MUST NOT compromise safety,
  isolation, or integrity to satisfy a failing test.
- Amendments MUST be proposed with a clear rationale and migration
  plan for affected code.
- Constitution versioning follows Semantic Versioning:
  - MAJOR: principle removal or backward-incompatible redefinition.
  - MINOR: new principle or materially expanded guidance.
  - PATCH: clarifications, wording, or non-semantic refinements.
- Compliance reviews MUST be performed as part of the self-audit
  requirement before every delivery.

**Version**: 1.0.0 | **Ratified**: 2026-02-22 | **Last Amended**: 2026-02-22
