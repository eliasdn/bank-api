## 2026-09-14 - Type mismatch bypasses user rate limiting
**Vulnerability:** A rate limiting bypass occurred due to a context key mismatch in middleware ("user_id" vs "userID") combined with an unsafe type assertion in the rate limiter `userID.(uint)`, which would panic or fail if the type changed (e.g. JWT parses numbers to float64, and RFC 7519 defines `sub` as string).
**Learning:** Gin Context keys are untyped. When multiple middlewares interact through `c.Set` and `c.Get`, mismatched keys lead to silent failures, and strict type assertions on claims from different origins introduce high regression risk.
**Prevention:** Use consistent constant variables for context keys, and write defensive type switches when parsing IDs from untyped sources like JWT token claims.

## 2026-09-15 - Mass Assignment Vulnerability in CreateAccount
**Vulnerability:** The `CreateAccount` API endpoint used `c.ShouldBindJSON(&account)` directly on the `models.Account` struct, allowing malicious users to arbitrarily set internal model fields, such as `Balance`, during account creation via a simple JSON payload (`{"balance": 1000000}`).
**Learning:** Directly binding HTTP JSON payloads to database ORM models exposes all public struct fields to mass assignment.
**Prevention:** Always use dedicated Request DTO structs to bind incoming JSON, explicitly defining which fields can be modified. Map the allowed DTO fields securely into the database models explicitly within the handler.

## 2026-09-15 - Hardcoded JWT Secret Key Fallback
**Vulnerability:** The application used a hardcoded fallback value (`"your-secret-key-change-this-in-production"`) for `JWT_SECRET` in `config.go` if the environment variable was missing. While the code enforced setting this variable in `production`, it allowed non-production environments to boot with this known secret. If a non-production instance was ever exposed or compromised, or if the environment variable check failed, this could lead to token forgery and full system compromise.
**Learning:** Hardcoding cryptographic secrets as fallbacks, even with conditional runtime checks for production environments, is a critical security risk. It creates a known backdoor and violates the principle of fail-secure design.
**Prevention:** Never use hardcoded strings for cryptographic keys or secrets. Always enforce the presence of required secrets at application startup. If a secret is missing, the application should fail securely (e.g., `log.Fatal`) rather than defaulting to an insecure state.
