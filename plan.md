1. **Add `uint_conversions.go` to `internal/validation`:**
   - Create highly optimized manual byte loop functions for `FormatUint(u uint64) string` and `ParseUint(s string, def uint64) (uint64, error)`.
   - These functions avoid generalized reflection/interface overhead of the `strconv` package, making them significantly faster in hot paths.
2. **Replace `strconv.FormatUint` usage:**
   - Update `internal/services/audit_service.go`
   - Update `internal/middleware/rate_limiter.go`
   - Update `internal/middleware/logging.go`
   - Update `internal/audit/audit.go`
   - Replace `strconv.FormatUint(uint64(id), 10)` with `validation.FormatUint(uint64(id))` in these files.
3. **Replace `strconv.ParseUint` usage:**
   - Update `internal/handlers/users.go`
   - Update `internal/middleware/auth.go`
   - Replace `strconv.ParseUint(idStr, 10, 64)` with `validation.ParseUint(idStr, 0)` in these files.
4. **Update tests and run verifications:**
   - Add unit/benchmark tests for the new conversion functions in `internal/validation/uint_conversions_test.go`.
   - Run `go test ./...` and `go build ./...` to verify everything compiles and passes.
5. **Complete pre commit steps**
   - Ensure proper testing, verification, review, and reflection are done by calling `pre_commit_instructions`.
6. **Submit PR:**
   - Submit the PR with the performance improvements.
