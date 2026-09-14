## 2024-05-22 - [Database Indexing Strategy]
**Learning:** SQLite supports `DESC` in index definitions, which is crucial for pagination queries that use `ORDER BY created_at DESC`. Without the `DESC` in the index, the database might still need to perform a sort operation or scan the index backwards (which is fast but explicit direction is better).
**Action:** When optimizing "latest items" lists, always prefer composite indexes `(foreign_key, sort_column DESC)` over simple foreign key indexes.
## Repeated Random Seeding Optimization

**Optimization**: Extracted `rand.Seed(time.Now().UnixNano())` out of `generateAccountNumber` into an `init()` function.
**Rationale**: Repeatedly seeding `math/rand` on every function call generates overhead and causes lock contention in highly concurrent environments because the global random generator is protected by a mutex. By moving it to `init()`, the seeding process happens only once during package initialization, improving throughput.
**Impact**: Performance benchmark demonstrated execution time improved from `420.2 ns/op` to `208.1 ns/op`, which makes it approximately two times faster.
## 2024-05-23 - [Regex Compilation in Hot Paths]
**Learning:** Recompiling regular expressions inside frequently called validation functions like `ValidatePassword` is a significant performance bottleneck. Replacing `regexp.MustCompile` with simple string operations like `strings.ContainsAny` for basic character class checks improves performance drastically (e.g., from ~6000 ns/op to ~175 ns/op).
**Action:** Avoid compiling regexes on the fly in hot paths. If regex is absolutely necessary, compile it once and store it in a package-level variable. Better yet, prefer faster alternatives like `strings.Contains` or `strings.ContainsAny` when validating basic character inclusions.

## 2024-05-23 - [Costly regexp initialization]
**Learning:** Initializing `regexp.MustCompile` inside validation functions causes the regex to be recompiled on every function call. This is incredibly inefficient for operations that happen frequently, such as user registrations or profile updates. For character presence checks, `strings.ContainsAny` is significantly faster (~17ms vs ~596ms for 100k iterations).
**Action:** Always prefer `strings.ContainsAny` or `strings.Contains` over regex for simple character inclusion checks. If regex is absolutely necessary, compile it at the package level as a global variable rather than instantiating it repeatedly inside functions.

## 2026-09-13 - [Fmt to Byte Slice Allocation Optimization]
**Learning:** `fmt.Sprintf` is consistently a high overhead source for simple, fixed-length string generation due to reflection.
**Action:** For performance sensitive generation like account numbers, generate string using fixed byte arrays avoiding `fmt`.
## Performance Optimization: Rate Limiter Cleanup Lock Contention
- **Date**: 2026-09-13
- **File**: `internal/middleware/rate_limiter.go`
- **Issue**: The `cleanupExpiredLimiters` function held an exclusive `sync.RWMutex.Lock()` for the entire duration of iterating over maps (`ipLimiters` and `userLimiters`) to find and delete expired rate limiters. For maps with large number of items (e.g., 100k IPs), this caused severe blocking and high latency for all incoming requests needing the rate limit middleware, which was on the critical path.
- **Solution**: Implemented a two-phase cleanup process:
  1. Acquire an `RLock()` to iterate over the maps safely and identify keys that are candidates for deletion.
  2. Store these candidate keys in a local slice.
  3. Release the `RLock()`.
  4. Only if candidates were found, acquire a full `Lock()`, iterate over the candidate slice, double check the deletion criteria (in case the item was used between releasing the `RLock` and acquiring the `Lock`), and perform the deletion.
- **Measured Improvement**: Benchmarking `BenchmarkRateLimiterConcurrentCleanup` with 100k items and concurrent read requests. Baseline latency dropped significantly by decoupling the map iteration (O(N) duration) from the exclusive lock scope. Worst-case locking per incoming request dropped drastically.

## 2024-05-18 - Replacing Regex with Byte Loops for Validation
**Learning:** For simple text validation in hot paths (like username formats), replacing `regexp.MustCompile` and `MatchString` with a manual byte loop can yield a ~30x performance improvement in Go (e.g. reducing time from ~550 ns/op to ~18 ns/op). This codebase prefers this optimization approach over regular expressions.
**Action:** When validating simple string formats containing alphanumeric characters or small sets of special characters, manually iterate through the string bytes instead of using regex.
## 2026-09-13 - Performance Optimization: Pagination Count Query
**Optimization**: Added in-memory caching using sync.Map for transaction count queries in GetTransactions.
**Rationale**: The Count query became an O(N) bottleneck for pagination on large accounts. Caching it reduces it to O(1) in the best case, with invalidation triggered on relevant writes (Deposit, Withdraw, Transfer).
**Impact**: BenchmarkGetTransactions improved execution time from ~44ms/op to ~2.5ms/op.

## 2024-05-24 - [Avoid Regex Recompilation in Loop]
**Learning:** `regexp.MustCompile` shouldn't be inside loops or file walk routines because the regex engine spends significant CPU cycles recompiling the same pattern for every iteration (e.g., every file parsed).
**Action:** Always move `regexp.MustCompile` statements to global or package-level variables so they are compiled exactly once at application startup.

## 2026-09-14 - [Avoid `strings.ContainsAny` in Hot Paths for Multiple Character Check]
**Learning:** Checking for the presence of character classes (uppercase, lowercase, numbers, specials) using multiple calls to `strings.ContainsAny` in a hot path like password validation is less efficient than a single manual byte loop. A single manual byte loop scans the string once and performs basic ASCII comparisons, avoiding the overhead of multiple function calls and inner loop executions within `strings.ContainsAny`.
**Action:** Replace multiple `strings.ContainsAny` checks with a single manual byte loop when validating simple string formats and character class requirements, especially in performance-sensitive parts of the application.
