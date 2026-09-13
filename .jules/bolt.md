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
