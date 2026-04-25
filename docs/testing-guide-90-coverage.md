# Testing Guide — Reach 90%+ Coverage (Practical, Non-Fake)

## 1) Target and Principles
- Target: package-level coverage `>= 90%` for critical packages (`internal/handler`, `internal/service`, `internal/middleware`).
- Do not chase coverage by testing only happy path.
- Every public method must have tests for:
  - success path
  - validation failure
  - authorization failure
  - dependency failure (repo/service returns error)
  - edge cases (`uuid.Nil`, empty input, malformed JSON, expired token)

## 2) Commands You Should Use Every Day
```bash
GOCACHE=/tmp/go-build go test ./... 
GOCACHE=/tmp/go-build go test -race ./...
GOCACHE=/tmp/go-build go test ./internal/handler ./internal/service ./internal/middleware -cover
GOCACHE=/tmp/go-build go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## 3) Coverage Strategy by Layer

### Handler tests (black-box style)
- Use `gin.CreateTestContext` + `httptest.NewRecorder`.
- Mock service interfaces; never hit DB in handler tests.
- Test matrix per endpoint:
  - unauthorized (missing `middleware.UserIDContextKey`)
  - invalid request body / invalid params
  - forbidden (`service.Err...Forbidden`)
  - not found
  - success response shape + status

### Service tests (business rules)
- Mock repository interfaces.
- Focus on domain rules, not HTTP status.
- Test matrix:
  - input validation (`uuid.Nil`, empty required fields)
  - authorization checks (membership false)
  - repo `ErrRecordNotFound` mapping
  - retry/idempotency behavior
  - race-safe logic if concurrent path exists

### Middleware tests
- Use real JWT signed in test with temporary RSA key.
- Test matrix:
  - missing token
  - invalid issuer/audience
  - expired token
  - linking-required flow
  - step-up required flow
  - success path (context keys are set correctly)

### Repository tests
- Use real test DB (PostgreSQL), not mocks.
- Verify query behavior with real constraints/indexes.
- Include tests for `ErrRecordNotFound`, unique violations, pagination boundaries.

## 4) Table-Driven Template (Recommended)
```go
func TestXxx(t *testing.T) {
  tests := []struct{
    name string
    input ...
    setup func(...)
    wantErr error
    wantCode int
  }{
    // success, bad input, forbidden, not found, internal error
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      // setup
      // execute
      // assert
    })
  }
}
```

## 5) Checklist Before You Say “Done”
- [ ] `go test ./...` passes
- [ ] `go test -race ./...` passes
- [ ] Critical packages `>= 90%`
- [ ] Unauthorized case exists for every protected endpoint
- [ ] Forbidden case exists where authorization is enforced
- [ ] Invalid input + edge UUID case exists
- [ ] No test relies on fragile error strings

## 6) How to Increase Coverage Fast (Without Fake Tests)
1. Run `go tool cover -func=coverage.out` and sort lowest packages first.
2. For each uncovered function, add branch tests in this order:
   - input validation branch
   - dependency error branch
   - authorization branch
   - success branch
3. Convert repetitive tests into table-driven tests.
4. Add one regression test for every bug you fix.

## 7) Anti-Patterns (Do Not Do)
- Only testing happy path.
- Asserting exact log strings.
- Writing tests that call internal private details instead of behavior.
- Mocking GORM in repository tests.
- Ignoring race detector failures.

