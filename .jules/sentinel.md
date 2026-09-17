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

## 2026-09-16 - Authentication Misconfiguration on Public Routes
**Vulnerability:** The application was improperly applying `authMiddleware.Authenticate()` to the `/api/v1/auth` route group in `cmd/server/main.go`, which includes the `/register` and `/login` endpoints. Since users require these exact endpoints to acquire an authentication token, enforcing token requirements here completely prevented any user from logging in or registering.
**Learning:** Security controls, such as authentication middlewares, must be deliberately applied only to protected routes. Blanketing all route groups with authentication, especially those explicitly designed for initial public access (like identity verification and onboarding), leads to a self-imposed denial-of-service condition for user authentication.
**Prevention:** The `/api/v1/auth` route group in `cmd/server/main.go` must remain public and exclude the `authMiddleware.Authenticate()` middleware to ensure users can acquire tokens. Always verify that login, registration, and initial onboarding routes are reachable without an active session token.
## 2026-09-16 - Prevent TOCTOU Race Conditions with Atomic SQL Updates
**Vulnerability:** The transaction endpoints (`Deposit`, `Withdraw`, `Transfer`) calculated new balances by reading the current balance, performing in-memory math, and writing a static value back. This created a severe Time-of-Check to Time-of-Use (TOCTOU) race condition where concurrent requests could overwrite each other, leading to double spending or lost deposits.
**Learning:** Using standard ORM assignment (`account.Balance = account.Balance - amount`) is not concurrency-safe in high-stakes environments without explicit locks.
**Prevention:** To prevent TOCTOU race conditions in financial transactions using GORM, use atomic SQL updates with conditions (e.g., `UpdateColumn("balance", gorm.Expr("balance - ?", amount))`) and verify `RowsAffected` rather than performing in-memory math.

## 2026-09-17 - Authentication denial for numeric string JWT subject claims
**Vulnerability:** Standard RFC 7519 JWT tokens containing string-formatted numeric subject claims (`"123"`) were stored as strings in Gin context, causing downstream handlers expecting `uint` user IDs to reject requests with `401 Unauthorized` or type assertion failures.
**Learning:** Middleware storing untyped claims in request contexts must parse string representations of numeric IDs into typed primitives before handing off to downstream handlers that rely on strong typing.
**Prevention:** In JWT authentication middleware, attempt `strconv.ParseUint` on string `sub` claims to set typed numeric values in context while retaining non-numeric strings for external identifiers.
## 2026-09-17 - Prevent username enumeration via timing attacks
**Vulnerability:** The login endpoint exhibited a timing attack vulnerability. If a username was not found in the database, the server returned an error immediately, skipping the computationally expensive `bcrypt` password check. Attackers could measure response times to enumerate valid usernames.
**Learning:** Returning early upon user lookup failure creates measurable timing differences that reveal whether an account exists, a classic username enumeration vector.
**Prevention:** To prevent username enumeration via timing attacks during authentication, login handlers must simulate password hashing (e.g., using `bcrypt.GenerateFromPassword`) when a user is not found, ensuring response times remain constant regardless of username validity.
