1. **Optimize UUID generation in logging middleware**
   - Replace `uuid.New().String()` with `uuid.NewString()` in `internal/middleware/logging.go` and `internal/middleware/error_logger.go`.
   - `uuid.NewString()` is the optimized standard way to generate a UUID string in Google's UUID library since v1.2.0. It eliminates intermediate `uuid.UUID` array copies on the stack when only the string format is needed.
   - This improves performance and reduces allocations in the hot path for every request that doesn't already have an `X-Request-ID`.

2. **Add an entry to `.jules/bolt.md`**
   - Document the use of `uuid.NewString()` over `uuid.New().String()` to avoid unnecessary struct initializations and method call overhead.

3. **Run tests**
   - Run `go test ./...` to verify the codebase is still correct.

4. **Complete pre-commit steps**
   - Follow instructions to make sure proper testing, verifications, reviews, and reflections are done before submit.

5. **Submit the PR**
   - Use the Bolt persona to create the PR, matching the 'Bolt: [performance improvement]' format and the required description elements.
