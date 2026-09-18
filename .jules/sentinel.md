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

## 2026-09-17 - Information Leakage via API Response
**Vulnerability:** The `Transfer` endpoint leaked the exact balance of the destination account in the API response `TransferResponse`. By making small transfers, any user could query the exact account balance of another user, resulting in a critical Information Leakage/Insecure Direct Object Reference (IDOR) vulnerability.
**Learning:** Returning struct models directly or over-sharing data in Response DTOs can easily leak sensitive information across user boundaries in multi-tenant or multi-user applications.
**Prevention:** Design Response DTOs carefully. When performing actions that affect resources owned by other users (like transfers), ensure the API response explicitly omits sensitive data about those third-party resources (such as `ToBalance`).

## 2026-09-17 - Orphaned active accounts on user profile deletion
**Vulnerability:** The `DeleteUser` handler previously soft-deleted the user record without cascading deletion to associated bank accounts or executing within a database transaction. This left orphaned bank accounts active in the database and failed to clear cached account counts.
**Learning:** GORM soft deletes do not automatically cascade across relationships unless explicitly executed in a transaction or handled at the database constraint level.
**Prevention:** When deleting primary user entities, always wrap deletion logic in a database transaction (`tx := db.Begin()`) that explicitly soft-deletes associated child resources (such as bank accounts) before deleting the parent user.

## 2026-09-18 - Password Hash Exposure in JSON Serialization
**Vulnerability:** The `User` struct's `PasswordHash` field lacked the `json:"-"` struct tag. When user objects were serialized to JSON (such as during audit logging of user registration, profile updates, and deletions), the bcrypt password hash was included in cleartext JSON in audit log entries.
**Learning:** Default Go JSON struct field tags serialize all public fields. Omitting `json:"-"` on sensitive credential fields allows confidential credentials/hashes to leak into logs, responses, or external systems.
**Prevention:** Always add `json:"-"` struct tags to sensitive credential and secret fields on data models to ensure they are excluded from automatic JSON serialization.

## 2026-09-19 - Non-Finite Floating Point Validation Bypass
**Vulnerability:** `ValidateAmount` checked `amount <= 0` and `amount > 1000000` but omitted non-finite floating-point checks. In IEEE 754 arithmetic (and Go float comparisons), comparisons against `math.NaN()` always evaluate to false, allowing `NaN` values to bypass range validation checks.
**Learning:** Standard comparison operators (`<=`, `>`) do not catch `NaN` values because all floating-point comparisons with `NaN` evaluate to `false`.
**Prevention:** Explicitly validate numeric inputs with `math.IsNaN(val)` and `math.IsInf(val, 0)` prior to relational range comparisons in critical financial or numeric validation paths.

## 2026-09-20 - Validation Bypass via Empty String in Struct JSON Deserialization
**Vulnerability:** Updating user profile allowed empty string (`""`) payloads for fields like `fullName` and `email` to bypass custom validation functions because non-pointer struct fields evaluate `""` as zero-value / unsupplied, while empty JSON payloads (`{}`) executed redundant DB write transactions and audit logging.
**Learning:** In Go Gin handlers, binding optional JSON fields to value types (like `string`) makes it impossible to distinguish between an omitted field and an explicitly passed empty string (`""`).
**Prevention:** Use pointer struct fields (`*string`) combined with Gin struct tags (`binding:"omitempty,min=2,max=100"`) and explicit checks (`FullName == nil && Email == nil`) to enforce presence and length constraints on partial updates.

## 2026-09-21 - Sub-Cent Precision Validation in Financial Transactions
**Vulnerability:** Transaction amount validation allowed floating-point values with more than 2 decimal places (fractional cents like $0.001 or $10.0001). This created a vulnerability to sub-cent micro-transaction flooding, salami-slicing attacks, and floating-point precision degradation in balance calculations.
**Learning:** Relational bounds checks (`gt=0`, `amount <= 1000000`) do not prevent sub-cent decimal values. In floating-point arithmetic, accumulating fractional cents can degrade balance precision and allow sub-cent manipulation.
**Prevention:** Enforce maximum 2 decimal places in currency amount validation using `math.Abs(amount*100-math.Round(amount*100)) > 1e-6` before processing financial operations.
