## 2026-09-14 - Type mismatch bypasses user rate limiting
**Vulnerability:** A rate limiting bypass occurred due to a context key mismatch in middleware ("user_id" vs "userID") combined with an unsafe type assertion in the rate limiter `userID.(uint)`, which would panic or fail if the type changed (e.g. JWT parses numbers to float64, and RFC 7519 defines `sub` as string).
**Learning:** Gin Context keys are untyped. When multiple middlewares interact through `c.Set` and `c.Get`, mismatched keys lead to silent failures, and strict type assertions on claims from different origins introduce high regression risk.
**Prevention:** Use consistent constant variables for context keys, and write defensive type switches when parsing IDs from untyped sources like JWT token claims.

## 2026-09-15 - Missing Strict Rate Limiting on Auth Endpoints
**Vulnerability:** The application was using the same generic rate limit for all endpoints (100 RPM per IP, 1000 RPH per user). Sensitive endpoints like `/auth/login` and `/auth/register` were susceptible to brute-forcing and credential stuffing attacks because 100 attempts per minute is far too generous for authentication.
**Learning:** Generic rate limits do not provide sufficient defense-in-depth for authentication endpoints. Threat actors can easily exploit high limit allowances.
**Prevention:** Implement endpoint-specific, strict rate limiting (e.g., 5 attempts per minute) for sensitive actions like authentication, password resets, and account registration. Separate these limiters into their own tracking structures to avoid interfering with general traffic.
## 2026-09-15 - Mass Assignment Vulnerability in CreateAccount
**Vulnerability:** The `CreateAccount` API endpoint used `c.ShouldBindJSON(&account)` directly on the `models.Account` struct, allowing malicious users to arbitrarily set internal model fields, such as `Balance`, during account creation via a simple JSON payload (`{"balance": 1000000}`).
**Learning:** Directly binding HTTP JSON payloads to database ORM models exposes all public struct fields to mass assignment.
**Prevention:** Always use dedicated Request DTO structs to bind incoming JSON, explicitly defining which fields can be modified. Map the allowed DTO fields securely into the database models explicitly within the handler.

## 2026-09-15 - Hardcoded JWT Secret Key Fallback
**Vulnerability:** The application used a hardcoded fallback value (`"your-secret-key-change-this-in-production"`) for `JWT_SECRET` in `config.go` if the environment variable was missing. While the code enforced setting this variable in `production`, it allowed non-production environments to boot with this known secret. If a non-production instance was ever exposed or compromised, or if the environment variable check failed, this could lead to token forgery and full system compromise.
**Learning:** Hardcoding cryptographic secrets as fallbacks, even with conditional runtime checks for production environments, is a critical security risk. It creates a known backdoor and violates the principle of fail-secure design.
**Prevention:** Never use hardcoded strings for cryptographic keys or secrets. Always enforce the presence of required secrets at application startup. If a secret is missing, the application should fail securely (e.g., `log.Fatal`) rather than defaulting to an insecure state.

## 2026-09-16 - Self-transfer balance duplication vulnerability
**Vulnerability:** The `Transfer` endpoint updated the source account balance (`fromAccount.Balance - amount`) and then the destination account balance (`toAccount.Balance + amount`) sequentially using the ORM. If the source and destination accounts were the same, the second update overwrote the first, effectively duplicating the transferred amount and artificially inflating the user's balance.
**Learning:** Sequential updates on the same database record using stale in-memory struct fields (e.g. `toAccount.Balance`) lead to logical flaws. When endpoints deal with transactions involving source and destination resources, failing to assert that the resources are distinct is a critical oversight.
**Prevention:** Always validate that source and destination resource IDs are distinct when processing transfers or similar operations (e.g. `if fromAccount.ID == transfer.ToAccountID { ... return error }`).
