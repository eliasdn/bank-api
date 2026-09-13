## 2024-05-22 - [Database Indexing Strategy]
**Learning:** SQLite supports `DESC` in index definitions, which is crucial for pagination queries that use `ORDER BY created_at DESC`. Without the `DESC` in the index, the database might still need to perform a sort operation or scan the index backwards (which is fast but explicit direction is better).
**Action:** When optimizing "latest items" lists, always prefer composite indexes `(foreign_key, sort_column DESC)` over simple foreign key indexes.
## Repeated Random Seeding Optimization

**Optimization**: Extracted `rand.Seed(time.Now().UnixNano())` out of `generateAccountNumber` into an `init()` function.
**Rationale**: Repeatedly seeding `math/rand` on every function call generates overhead and causes lock contention in highly concurrent environments because the global random generator is protected by a mutex. By moving it to `init()`, the seeding process happens only once during package initialization, improving throughput.
**Impact**: Performance benchmark demonstrated execution time improved from `420.2 ns/op` to `208.1 ns/op`, which makes it approximately two times faster.
## 2024-05-23 - [Costly regexp initialization]
**Learning:** Initializing `regexp.MustCompile` inside validation functions causes the regex to be recompiled on every function call. This is incredibly inefficient for operations that happen frequently, such as user registrations or profile updates. For character presence checks, `strings.ContainsAny` is significantly faster (~17ms vs ~596ms for 100k iterations).
**Action:** Always prefer `strings.ContainsAny` or `strings.Contains` over regex for simple character inclusion checks. If regex is absolutely necessary, compile it at the package level as a global variable rather than instantiating it repeatedly inside functions.
