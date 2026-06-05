# Phase 9: Middleware & Integration Tests — Execution Plan

> **Track**: Track 1 — Testing  
> **Phase**: 9 of 9  
> **Status**: 🟡 Planned

---

## Scope Overview

Phase 9 closes the testing track with two deliverables:

1. **JWT Middleware Tests** — unit tests for `VerifyToken` in `internal/api/middleware/middleware.go` using `github.com/danielgtaylor/huma/v2/humatest` — a sub-package of the already-imported `huma/v2` module. No new dependency, no interface mocking.
2. **CI Coverage Gate** — a new `make coverage-check` Makefile target that generates a coverage profile, computes the total coverage, and fails if it is below **80%**; the existing `ci.yml` is updated to call it.

**Explicitly Out of Scope**:
- Role-checking middleware tests (Phase 9 item 2) are **deferred to Track 3** when the role middleware is actually implemented. No placeholder or stub is created — the plan item is marked as blocked with a cross-reference.

---

## Decisions (from User Clarification)

| Decision Point | Choice |
|---|---|
| Role-checking middleware (item 2) | Skip entirely; deferred to Track 3 |
| Middleware test approach | `humatest.New()` — real huma API + `httptest.ResponseRecorder`; zero mocking |
| CI coverage gate mechanism | `make coverage-check` with `awk` threshold check; called from `ci.yml` |
| Middleware test file location | `internal/api/middleware/middleware_test.go` (`package middleware_test`, external) |
| Role middleware in plan doc | Marked explicitly out of scope with Track 3 cross-reference |

---

## Deliverable 1 — JWT Middleware Tests

### File

```
internal/api/middleware/middleware_test.go
```

### Package

```go
package middleware_test
```

External test package — no need for whitebox access since `humatest` drives the middleware through real HTTP requests.

### Strategy: `humatest.New()` — Zero Mocking, Real HTTP

`github.com/danielgtaylor/huma/v2/humatest` is a **sub-package of the already-imported `huma/v2` module** (confirmed in `go.mod`). It requires **no new dependency** and is the approach officially recommended by the Huma maintainers for middleware testing.

The pattern for each test case:

1. **Create a test API** — `humatest.New(t, huma.DefaultConfig(...))` returns `(http.Handler, humatest.TestAPI)`.
2. **Register the middleware** — `testAPI.UseMiddleware(VerifyToken(testAPI))`.
3. **Register a sentinel route** — a minimal `GET /test` handler that records whether it was reached.
4. **Fire a request** via `testAPI.Do("GET", "/test", "Cookie: authToken=<value>")`, which returns `*httptest.ResponseRecorder`.
5. **Assert on `resp.Code`** — the HTTP status code written by the middleware or the sentinel handler.

No mock structs. No interface stubs. The real `huma.ReadCookie`, `huma.WriteErr`, and `huma.WithContext` all execute against the real humatest context implementation.

#### Sentinel Handler

A shared no-op handler is registered on the test API to confirm the middleware called `next`:

```go
type emptyInput struct{}
type emptyOutput struct{}

huma.Register(testAPI, huma.Operation{
    Method:      http.MethodGet,
    Path:        "/test",
    OperationID: "test-sentinel",
}, func(ctx context.Context, input *emptyInput) (*emptyOutput, error) {
    return &emptyOutput{}, nil
})
```

When the middleware short-circuits (401 / 403), the handler is never reached and `resp.Code` reflects the middleware's written status.

#### Token Generation

Tokens are produced with `utils.GenerateJWT` directly. Each test case passes the signing secret explicitly to `VerifyToken`, so `config.GlobalConfig` is **never mutated** — the tests are fully stateless and safe to run in parallel.

#### `VerifyToken` Signature Note

The current signature is:
```go
func VerifyToken(api huma.API) func(ctx huma.Context, next func(huma.Context))
```
`humatest.TestAPI` embeds `huma.API`, so passing `testAPI` directly satisfies the parameter. No adapter needed.

---

### Test Cases

The suite is named **`MiddlewareTestSuite`** with a single exported runner `TestMiddlewareSuite(t *testing.T)`.

#### `TestVerifyToken` — table-driven subtests via `s.Run()`

| # | Test Name | Cookie Header | JWT Valid? | Expected `resp.Code` |
|---|---|---|---|---|
| 1 | `Success — valid token` | `authToken=<valid JWT>` | YES | `204 No Content` (sentinel reached) |
| 2 | `Failure — missing authToken cookie` | *(none)* | — | `401 Unauthorized` |
| 3 | `Failure — expired JWT` | `authToken=<expired JWT>` | NO (expired) | `403 Forbidden` |
| 4 | `Failure — malformed token string` | `authToken=not.a.jwt` | NO (malformed) | `403 Forbidden` |
| 5 | `Failure — wrong secret` | `authToken=<JWT signed with wrong secret>` | NO (bad sig) | `403 Forbidden` |

> The sentinel handler returns an empty `204` body. `humatest` uses `httptest.ResponseRecorder` so `resp.Code` is always populated even without a real TCP connection.

#### Suite Setup

A fresh `testAPI` and sentinel route are created in `SetupTest` so each subtest gets an isolated API instance:

```go
type MiddlewareTestSuite struct {
    suite.Suite
    testAPI humatest.TestAPI
    secret  string
}

func (s *MiddlewareTestSuite) SetupTest() {
    s.secret = "test-secret"
    _, s.testAPI = humatest.New(s.T(), huma.DefaultConfig("Test API", "1.0.0"))
    s.testAPI.UseMiddleware(VerifyToken(s.testAPI))
    huma.Register(s.testAPI, huma.Operation{
        Method:      http.MethodGet,
        Path:        "/test",
        OperationID: "test-sentinel",
    }, func(ctx context.Context, input *struct{}) (*struct{}, error) {
        return nil, nil
    })
}
```

Each subtest calls `s.SetupTest()` explicitly at the top of its `s.Run` body to get a clean slate.

---

### Test Suite File Structure

```go
package middleware_test

import (
    "context"
    "net/http"
    "testing"
    "time"

    middleware "lorem-backend/internal/api/middleware"
    "lorem-backend/internal/utils"

    "github.com/danielgtaylor/huma/v2"
    "github.com/danielgtaylor/huma/v2/humatest"
    "github.com/google/uuid"
    "github.com/stretchr/testify/suite"
)

// MiddlewareTestSuite tests the VerifyToken middleware via humatest.
type MiddlewareTestSuite struct {
    suite.Suite
    testAPI humatest.TestAPI
    secret  string
}

func (s *MiddlewareTestSuite) SetupTest() { ... }
func (s *MiddlewareTestSuite) TestVerifyToken() { ... }

func TestMiddlewareSuite(t *testing.T) {
    suite.Run(t, new(MiddlewareTestSuite))
}
```

---

## Deliverable 2 — CI Coverage Gate

### 2a. Makefile Changes

Add a `coverage-check` target to `Makefile`:

```makefile
coverage-check:
	go test -coverprofile=coverage.out ./internal/...
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | tr -d '%'); \
	echo "Total coverage: $$COVERAGE%"; \
	awk "BEGIN{if ($$COVERAGE + 0 < 80) {print \"FAIL: Coverage \" $$COVERAGE \"% is below 80% threshold\"; exit 1}}"
	@rm -f coverage.out
```

> NOTE: The `awk BEGIN` comparison approach is used to avoid a `bc` dependency, keeping this compatible with all standard CI runners including `ubuntu-latest`.

### 2b. CI Workflow Changes (`ci.yml`)

Update the `test` job to replace `make test` with `make coverage-check`:

```yaml
- name: Run Tests with Coverage Gate
  run: make coverage-check
```

The existing `make test` step is **replaced** by `make coverage-check` since coverage-check already runs all tests. The `lint` and `build-docker` jobs are **unchanged**.

#### Updated `ci.yml` `test` job (after change):

```yaml
test:
  name: Run Unit Tests
  runs-on: ubuntu-latest
  steps:
    - name: Checkout Code
      uses: actions/checkout@v4

    - name: Setup Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.25'

    - name: Run Tests with Coverage Gate
      run: make coverage-check
```

---

## File Change Summary

| File | Action | Description |
|---|---|---|
| `internal/api/middleware/middleware_test.go` | **Create** | New test file (`package middleware_test`) with `MiddlewareTestSuite` and 5 JWT middleware test cases driven by `humatest` |
| `Makefile` | **Edit** | Add `coverage-check` target |
| `.github/workflows/ci.yml` | **Edit** | Replace `make test` with `make coverage-check` in the `test` job |

---

## Implementation Order

1. **Write `middleware_test.go`** — Implement `MiddlewareTestSuite` using `humatest.New()` with all 5 test cases. No mocks to design or stub.
2. **Run `go test ./internal/api/middleware/...`** locally — Verify all 5 tests pass and coverage of `middleware.go` is >=80%.
3. **Add `coverage-check` to `Makefile`** — Implement the target with threshold check.
4. **Update `ci.yml`** — Replace `make test` with `make coverage-check`.
5. **Run `go vet ./...` and `gofmt` check** — Confirm zero issues.
6. **Run `make coverage-check`** locally — Confirm overall project coverage meets >=80%.

---

## Validation Checklist (from `validation.md`)

- [ ] `go test ./internal/modules/... -v` — zero failures, zero skipped
- [ ] `go test ./internal/api/... -v` — zero failures (middleware tests)
- [ ] Suite pattern enforced: `middleware_test.go` has exactly **one** exported `func Test*`
- [ ] `go vet ./...` — zero issues
- [ ] `gofmt` — zero formatting issues
- [ ] `make coverage-check` exits 0 (overall >=80%)
- [ ] CI `test` job uses `make coverage-check` and fails if threshold is breached
- [ ] Role-checking middleware tests explicitly marked as **out of scope** (deferred to Track 3)

---

## Out of Scope (Deferred)

**Role-Checking Middleware Tests (Plan item 2)** are explicitly out of scope for Phase 9.

The role-checking middleware is currently a `// TODO: Implement Role Checking Middleware` comment in `middleware.go`. No implementation exists. Tests cannot be written until the middleware is implemented in **Track 3**.

Cross-reference: This will be revisited as the **first item** in the Track 3 testing sub-phase once the role middleware is implemented.
