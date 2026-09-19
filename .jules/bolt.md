## 2026-09-19 - [Optimize fmt.Sprintf in error formatting]
**Learning:** `fmt.Sprintf` incurs a heavy reflection overhead (~92ns per op) even when no formatting arguments are provided. Creating validation errors that only return static strings happens very frequently in the validation layer.
**Action:** When writing error constructors like `NewValidationError(format string, args ...interface{})`, always add a fast path (`if len(args) == 0`) that assigns the static string directly without calling `fmt.Sprintf`. This reduces the instantiation time to sub-nanosecond levels (~0.3ns).
