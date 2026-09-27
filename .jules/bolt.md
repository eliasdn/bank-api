## 2026-09-19 - [Optimize fmt.Sprintf in error formatting]
**Learning:** `fmt.Sprintf` incurs a heavy reflection overhead (~92ns per op) even when no formatting arguments are provided. Creating validation errors that only return static strings happens very frequently in the validation layer.
**Action:** When writing error constructors like `NewValidationError(format string, args ...interface{})`, always add a fast path (`if len(args) == 0`) that assigns the static string directly without calling `fmt.Sprintf`. This reduces the instantiation time to sub-nanosecond levels (~0.3ns).
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
## 2026-09-14 - [Fmt.Sprintf integer string formatting]
**Learning:** `fmt.Sprintf` incurs reflection overhead for simple type formatting such as converting `uint` to `string`. For `ResourceID` generation, `fmt.Sprintf("%d", id)` was taking ~116ms per million iterations, while `strconv.FormatUint(uint64(id), 10)` took only ~39ms.
**Action:** Replace `fmt.Sprintf` with `strconv` formatting functions (e.g., `strconv.FormatUint`) when converting numbers to strings where performance matters, avoiding the reflection overhead of `fmt`.
## 2023-10-27 - [Optimize Password Validation]
**Learning:** Re-iterating over the same string with multiple `strings.ContainsAny` calls in a validation loop is inefficient and creates a bottleneck. A single pass using a manual byte loop to set booleans, combined with `strings.IndexByte` for a predefined special character set, significantly reduces execution time (from ~180 ns/op to ~42 ns/op).
**Action:** Consolidate multiple validation checks into a single byte loop where possible. Prefer `strings.IndexByte` over `strings.ContainsAny` for single-character set checks during a manual loop to maximize performance while retaining readability.

## 2026-09-14 - [Balancing Micro-Optimizations and Readability]
**Learning:** When replacing `strings.ContainsAny` with a manual byte loop to avoid multiple iterations over a string, using a massive `switch` case for checking characters against a set of special characters severely impacts readability.
**Action:** Use `strings.IndexByte("!@#$...", c) >= 0` within the manual byte loop to efficiently check if a character belongs to a specific set. This maintains the performance benefit of a single-pass loop while keeping the code concise and readable.
## 2026-09-14 - [Avoid `strings.ContainsAny` in Hot Paths for Multiple Character Check]
**Learning:** Checking for the presence of character classes (uppercase, lowercase, numbers, specials) using multiple calls to `strings.ContainsAny` in a hot path like password validation is less efficient than a single manual byte loop. A single manual byte loop scans the string once and performs basic ASCII comparisons, avoiding the overhead of multiple function calls and inner loop executions within `strings.ContainsAny`.
**Action:** Replace multiple `strings.ContainsAny` checks with a single manual byte loop when validating simple string formats and character class requirements, especially in performance-sensitive parts of the application.
## 2026-09-16 - [Replace string(rune(int)) with strconv.FormatUint]
**Learning:** Casting a numeric ID (like `uint`) to `rune` and then to `string` using `string(rune(id))` correctly interprets the numeric ID as a Unicode code point, producing the corresponding character (e.g. `string(rune(65))` results in `"A"`), which is almost certainly not the intended behavior when trying to log a numeric string like `"65"`. This is both a logic bug and a performance bottleneck if used in logging or hot paths. Using `strconv.FormatUint(uint64(id), 10)` is the correct and performant way to convert integers to their string representations without using reflection or accidentally getting Unicode characters.
**Action:** Always replace `string(rune(id))` with `strconv.FormatUint` (or `strconv.Itoa`/`strconv.FormatInt`) when the goal is to convert an integer ID to a string.
## 2026-09-16 - [Optimize formatting floats]
**Learning:** For performance sensitive code paths, using string concatenation alongside `strconv.FormatFloat` can provide a performance benefit over `fmt.Sprintf` due to eliminating reflection overhead.
**Action:** In places that run frequently, replace `fmt.Sprintf("%s %.2f", str, float)` with string concatenation and `strconv.FormatFloat(float, 'f', 2, 64)`

## 2023-09-15 - Fast Path Email Validation
**Learning:** `regexp.MustCompile` overhead for simple matching in hot paths like `ValidateEmail` takes around ~811ns per op. Using a manual byte loop to validate email formats reduces the overhead by ~94%, dropping execution time to ~46ns.
**Action:** When performing format validation, especially in frequently executed validation rules, prefer manual byte loops over regular expressions to maximize performance.

## 2026-09-15 - [Audit Log Transaction Serialization]
**Learning:** In the audit logging package, serializing transaction logs using `json.Marshal(map[string]interface{}{...})` is significantly slower than marshaling a dedicated struct due to the reflection overhead mapping and map allocation. Furthermore, `fmt.Sprintf` incurs reflection overhead for simple string concatenations when formatting floats or integers into strings.
**Action:** Always prefer using a dedicated struct for JSON serialization rather than `map[string]interface{}`. Use string concatenation alongside `strconv.FormatFloat` or `strconv.FormatUint` instead of `fmt.Sprintf` for constructing strings from simple primitive values in hot paths.
## 2024-05-18 - [Email Validation Performance Boost]
**Learning:** Using `regexp.MustCompile` and `MatchString` for string validations in hot paths adds significant performance overhead. A single manual byte loop check for email validation can be up to 10x faster. Additionally, duplicating logic leads to unoptimized methods being used when optimized versions already exist elsewhere in the codebase.
**Action:** Always check if a highly-optimized manual check exists centrally (like in `internal/validation/validation.go`) before resorting to regular expressions. Remove unused regex compilations to save memory and initialization time.
## 2026-09-16 - Zero Allocation String Set Validation
**Learning:** Initializing a local map `map[string]bool{...}` in a function to check a string against a fixed set of values is a performance anti-pattern in Go, as it forces heap allocation and population of the map on every function call.
**Action:** Always use a `switch` statement for fixed set string validation. It compiles down to fast string comparisons with exactly zero allocations.
## 2026-09-17 - [Eliminating reflection overhead in error formatting]
**Learning:** For formatting strings that involve errors or simple concatenation in hot paths, using `fmt.Sprintf("%s: %v", ...)` introduces unnecessary runtime reflection overhead due to the `%v` verb and internal parsing.
**Action:** Always prefer native string concatenation (e.g., `e.Message + ": " + e.Err.Error()`) over `fmt.Sprintf` for simple combinations of strings and errors to improve performance and reduce allocations.
## 2026-09-17 - Performance Optimization: Pagination Account Count Query
**Optimization**: Added in-memory caching using sync.Map for account count queries in GetAccounts.
**Rationale**: The Count query became an O(N) bottleneck for pagination on user accounts, similar to transactions count bottleneck. Caching it reduces it to O(1) in the best case, with invalidation triggered on relevant writes (CreateAccount).
**Impact**: BenchmarkGetAccounts improved execution time from ~1059ms/op to ~0.9ms/op for users with 1000 accounts.
## 2026-09-17 - Optimize Cache Key in GetTransactions
**Learning:** Passing raw URL parameters as strings to cache keys introduces overhead from integer parsing (`strconv.ParseUint`) and reflection when a database model object (like `account.ID` of type `uint`) has already been fetched and can be used directly.
**Action:** Always reuse strongly-typed fields from already-fetched GORM models instead of re-parsing string parameters for operations like caching.
## 2026-09-17 - DRY Performance Optimization
**Learning:** When duplicating highly-optimized methods (like custom manual byte loops) across packages (e.g., in handlers and validation packages), we can introduce unused imports or broken benchmarks when we try to clean it up. Keeping performance-optimized functions centralized in one logical package (like `internal/validation/validation.go`) prevents these issues.
**Action:** Always centralize optimized logic and reuse it across the application instead of duplicating it. When performing deduplication refactors, make sure to clean up any related benchmarks or test references in the removed locations and migrate them to the centralized location.
## 2026-09-17 - Performance Optimization: Query Parameter Parsing
**Optimization**: Replaced `strconv.Atoi` with a custom manual byte loop function `validation.ParseInt` for parsing string query parameters (e.g., page, limit) into integers.
**Rationale**: `strconv.Atoi` overhead is unnecessary for parsing simple integer strings from query parameters, especially with fallback default values. A simple manual byte loop eliminates function overhead and provides faster parsing, avoiding reflection-like overhead for small positive integers.
**Impact**: Reduced simple integer parsing time by ~20%.
## 2025-02-18 - Rate Limiter Map Key Optimization
**Learning:** In high-throughput middleware like the user rate limiter (`internal/middleware/rate_limiter.go`), dynamically converting generic context values (like `userID` which could be `uint`, `float64`, or `string`) into strings for map keys using a type switch and `strconv` incurs significant allocation overhead. However, attempting to strictly parse it as `uint` using Gin's `c.GetUint()` is dangerous and can silently drop non-uint values, leading to rate limit bypasses.
**Action:** Use a `map[interface{}]*rate.Limiter` instead of `map[string]*rate.Limiter`. This allows passing the generic context value directly into the map lookup (since the underlying types are inherently comparable), which preserves type safety across various implementations without the overhead of dynamic string formatting. This reduces lookup time from ~60ns to ~22ns.
## 2026-09-19 - Rate Limiter Map Key Optimization
**Learning:** In high-throughput middleware like the user rate limiter (`internal/middleware/rate_limiter.go`), dynamically converting generic context values (like `userID` which could be `uint`, `float64`, or `string`) into strings for map keys using a type switch and `strconv` incurs significant allocation overhead. However, attempting to strictly parse it as `uint` using Gin's `c.GetUint()` is dangerous and can silently drop non-uint values, leading to rate limit bypasses.
**Action:** Use a `map[interface{}]*rate.Limiter` instead of `map[string]*rate.Limiter`. This allows passing the generic context value directly into the map lookup (since the underlying types are inherently comparable), which preserves type safety across various implementations without the overhead of dynamic string formatting. This reduces lookup time from ~60ns to ~22ns.
## 2026-09-19 - Optimize Account Number Generation
**Learning:** Using `crypto/rand.Int` alongside `math/big.NewInt` for simple random number generation within a bounded range introduces significant allocation overhead and execution time (~1450 ns/op) compared to reading raw bytes and doing manual modular arithmetic.
**Action:** When a cryptographically secure random number is required within a small range (e.g., generating an account number between 0 and 999999), use `crypto/rand.Read` to fetch 4 bytes into a `[4]byte` array, convert it to a `uint32`, and apply modulo arithmetic (`%`). This maintains security while reducing execution time to ~1200 ns/op and eliminating `math/big` heap allocations.
## 2026-09-17 - Performance Optimization: Database Indexes
**Optimization**: Added `gorm:"index"` tags to `UserID` in the `Account` struct and `AccountID` in the `Transaction` struct.
**Rationale**: These fields are frequently queried in the handlers (e.g., `GetAccounts` filters by `user_id`, `GetTransactions` filters by `account_id`). Without indexes, these queries require full table scans, which become progressively slower as the number of records grows. Adding indexes allows the database to perform O(log N) or O(1) lookups.
**Impact**: Substantially improves lookup query performance for accounts and transactions, ensuring predictable response times as the dataset scales.
## 2025-02-18 - [Eliminate strings.ToLower allocation in dual-case validation loops]
**Learning:** Using `strings.ToLower` on a string prior to validating it with a manual byte loop that already explicitly checks for both uppercase (`A-Z`) and lowercase (`a-z`) characters introduces unnecessary O(N) allocation and conversion overhead. In the case of `ValidateEmail`, this allocation was responsible for ~70% of the function's execution time (~160ns/op down to ~46ns/op after removal).
**Action:** When writing or reviewing custom manual byte validation loops, check if the loop already safely handles both character cases. If so, remove any preceding `strings.ToLower` or `strings.ToUpper` normalization calls to avoid heap allocations and improve performance in hot paths.
## 2026-09-21 - [Zero-Value Redundant Pass Validation]
**Learning:** In `internal/validation/validation.go`, `ParseInt` avoids redundant O(N) validation passes for zero-value results because the primary parsing loop already safely rejects non-digit characters, improving performance for zero-valued strings. The second `allZeros` check is unnecessary.
**Action:** When implementing or reviewing custom string parsing loops, ensure that edge cases like zero are properly handled by the initial condition checks and primary loop, eliminating the need for subsequent redundant O(N) passes.

## 2026-09-20 - [Deduplicate Integer Parsing Logic]
**Learning:** Duplicating highly-optimized manual byte loops (like ParseInt and ParsePaginationParam) across packages or within the same package leads to maintenance overhead, redundant tests, and increased binary size without any additional performance benefit. When performing deduplication refactors, ensure that any related benchmarks, test references, and unused imports in the modified files are also cleaned up or migrated to the new centralized location.
**Action:** Always centralize optimized logic (such as string-to-integer parsing) and reuse it across the application instead of duplicating it. Clean up associated redundant tests (like FuzzParsePaginationParam) to align with DRY principles.
## 2026-09-22 - [Eliminate strings.IndexByte overhead in byte validation loops]
**Learning:** Using `strings.IndexByte` within a hot manual byte loop for checking characters against a fixed string set (like special characters in password validation) introduces function call overhead that can be avoided. A direct switch case or logical OR condition compiles down to more efficient instructions, improving execution time by ~30% (~35ns/op down to ~23ns/op).
**Action:** Always prefer explicit logical OR (`||`) character comparisons or switch cases over helper functions like `strings.IndexByte` or `strings.ContainsAny` inside tight, manual byte validation loops for bounded character sets.
## 2026-09-21 - [Optimize Secure Random Generation Without Modulo Bias]
**Learning:** Replacing `crypto/rand.Int` with `crypto/rand.Read` and manual modulo arithmetic (`%`) is a great way to avoid `math/big` heap allocations, but naively applying modulo to a `uint32` introduces **modulo bias**, weakening the uniform distribution required for cryptographically secure random number generation.
**Action:** When optimizing secure random generation, you must combine `crypto/rand.Read` and binary conversion with a **rejection sampling** loop. Calculate the maximum acceptable threshold (`maxAcceptable = 2^32 - (2^32 % range)`) and reject/re-roll any value greater than or equal to this threshold to guarantee a perfectly uniform distribution without sacrificing performance.
## 2026-09-23 - [Pre-allocate byte slices in middleware]
**Learning:** In high-throughput middleware, converting a static string config value to a byte slice inside the request handler (e.g., `[]byte(m.config.JWT.Secret)`) causes unnecessary heap allocations per request. Additionally, using standard string manipulation functions like `strings.TrimPrefix` often involves internal function call overhead and can sometimes allocate memory under specific conditions.
**Action:** When a configuration value needs to be passed as a byte slice to an underlying library, convert and cache it as a `[]byte` during the struct initialization (e.g., `NewAuthMiddleware`). Furthermore, replace common string helper functions with direct bounds-checked slicing and string comparisons (e.g., `if len(s) >= 7 && s[:7] == "Bearer "`) in tight execution paths to achieve zero-allocation performance and avoid function call overhead.
## 2026-09-24 - [Replace RWMutex with sync.Map in High-Throughput Read Paths]
**Learning:** In highly concurrent, read-heavy scenarios like the middleware rate limiter (`internal/middleware/rate_limiter.go`), using a standard `map` combined with `sync.RWMutex` introduces significant lock contention overhead.
**Action:** When implementing or optimizing high-throughput concurrent maps where keys are mostly read and rarely created/deleted, replace `map` + `sync.RWMutex` with `sync.Map`. This leverages lock-free reads and optimizes concurrent execution, reducing operation time from ~115ns down to ~39ns.
## 2026-09-24 - [Skip empty DB queries on out-of-bounds pagination]
**Learning:** When retrieving paginated data, if the requested offset (derived from page and limit) is greater than or equal to the total number of records, querying the database will yield zero results. Proceeding with a paginated `Find` query in this state introduces unnecessary database network I/O and query execution overhead.
**Action:** Always check `if int64(offset) < total` before executing the data fetch query when the total count is already available (especially when cached). If the offset is out-of-bounds, skip the query and return an empty list directly.
## 2026-09-25 - [Optimize Struct Iteration Overhead in Loops]
**Learning:** Iterating over a slice of structs using value semantics (`for i, val := range slice`) causes the Go runtime to copy the entire struct into the `val` variable on every iteration. For large structs like database models, this memory copying adds unnecessary overhead, particularly in hot paths or with large data sets (e.g., retrieving paginated transactions).
**Action:** When iterating over a slice of structs, always use index semantics (`for i := range slice`) and access the struct via a pointer `&slice[i]` or directly by index `slice[i]`. This completely avoids the struct copying overhead and improves performance.
## 2026-09-26 - [Optimize UUID generation]
**Learning:** Generating a UUID and converting it to a string using `uuid.New().String()` creates an intermediate `uuid.UUID` array (16 bytes) and then calls its `.String()` method, causing overhead. The google/uuid library provides `uuid.NewString()` which is optimized to do this directly, reducing execution time.
**Action:** When only the string representation of a random UUID is needed, prefer using `uuid.NewString()` over `uuid.New().String()` to optimize hot paths.
## 2026-09-27 - Performance Optimization: Float formatting in hot paths
**Learning**: In `internal/audit/audit.go`, string formatting for transaction amounts used `strconv.FormatFloat`. For specific fixed-precision financial formats (like exactly 2 decimal places for cents), a custom manual byte loop avoids standard library overhead.
**Action**: Replaced `strconv.FormatFloat(amount, 'f', 2, 64)` with a highly optimized `validation.FormatFloatAmount` function to format `float64` values into strings faster in hot paths.
