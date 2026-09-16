## 2026-09-14 - Type mismatch bypasses user rate limiting
**Vulnerability:** A rate limiting bypass occurred due to a context key mismatch in middleware ("user_id" vs "userID") combined with an unsafe type assertion in the rate limiter `userID.(uint)`, which would panic or fail if the type changed (e.g. JWT parses numbers to float64, and RFC 7519 defines `sub` as string).
**Learning:** Gin Context keys are untyped. When multiple middlewares interact through `c.Set` and `c.Get`, mismatched keys lead to silent failures, and strict type assertions on claims from different origins introduce high regression risk.
**Prevention:** Use consistent constant variables for context keys, and write defensive type switches when parsing IDs from untyped sources like JWT token claims.

## 2026-09-15 - Missing Strict Rate Limiting on Auth Endpoints
**Vulnerability:** The application was using the same generic rate limit for all endpoints (100 RPM per IP, 1000 RPH per user). Sensitive endpoints like `/auth/login` and `/auth/register` were susceptible to brute-forcing and credential stuffing attacks because 100 attempts per minute is far too generous for authentication.
**Learning:** Generic rate limits do not provide sufficient defense-in-depth for authentication endpoints. Threat actors can easily exploit high limit allowances.
**Prevention:** Implement endpoint-specific, strict rate limiting (e.g., 5 attempts per minute) for sensitive actions like authentication, password resets, and account registration. Separate these limiters into their own tracking structures to avoid interfering with general traffic.
