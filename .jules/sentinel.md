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

## 2026-09-22 - CRLF and Null-Byte Injection in Transaction Descriptions
**Vulnerability:** Transaction description validation previously checked length (`MaxDescriptionLength = 255`) but permitted null bytes (`\x00`), carriage returns (`\r`), and newlines (`\n`). This exposed application audit logs to CRLF log injection and SQLite parameter binding to potential null-byte truncation.
**Learning:** Length checks alone do not prevent control character or null-byte injection vulnerabilities when text fields are persisted in database logs or written to structured application logs.
**Prevention:** Reject null bytes (`\x00`), carriage returns (`\r`), and newlines (`\n`) during input validation for free-text parameters using byte loops prior to logging or storing them.

## 2026-09-23 - CRLF and Null-Byte Injection in User Full Name Validation
**Vulnerability:** User full name validation previously checked length constraints (`MinFullNameLength = 2`, `MaxFullNameLength = 100`) and non-emptiness, but permitted null bytes (`\x00`), carriage returns (`\r`), and newlines (`\n`). This exposed user registration and profile update audit logs to CRLF log injection and potential null-byte truncation.
**Learning:** Checking length and whitespace presence alone is insufficient for user name inputs when those inputs are written to structured application and audit logs.
**Prevention:** Reject null bytes (`\x00`), carriage returns (`\r`), and newlines (`\n`) using byte loops during input validation for full name fields prior to storing or logging them.

## 2026-09-21 - Prevent Rate-Limit Bypass via Context Type Mismatch
**Vulnerability:** Middlewares like rate limiters use values from the request context (like `userID`) for tracking. `Gin` context values are loosely typed (`interface{}`). When upstream authentication middleware (like `auth.go` using JWTs) stored the `sub` claim as varying types (`string`, `float64`, `uint`), downstream middlewares or handlers calling strict getters like `c.GetUint("userID")` would fail type assertions and return `0`. This caused all such requests to share the `0` rate limit bucket (or bypass user limits entirely) while potentially still executing business logic if downstream handlers didn't fail securely, or caused false negatives during authorization checks.
**Learning:** Using strict type getters (e.g. `c.GetUint()`) on loosely typed context frameworks without defensive type checking creates severe regressions if upstream middlewares change the type of the value being stored, silently dropping identifiers and creating logic bypasses.
**Prevention:** Always use safe helper functions to extract numeric IDs from contexts (e.g., a `getUserID` helper using a type switch for `uint`, `float64`, and `string`) rather than relying on strict framework type assertions.
## 2026-09-24 - IP Spoofing and Rate Limit Bypass via X-Forwarded-For
**Vulnerability:** The application used `gin.Default()` without configuring trusted proxies. By default, older versions of Gin blindly trust the `X-Forwarded-For` and `X-Real-IP` headers. This allowed attackers to easily spoof their IP address by injecting these headers, completely bypassing IP-based rate limiting (e.g., on brute-force susceptible login endpoints) and falsifying audit logs.
**Learning:** Default settings in web frameworks regarding proxy headers are often insecurely permissive for convenience, but they expose applications directly connected to the internet to IP spoofing attacks.
**Prevention:** Always explicitly configure trusted proxies for web frameworks. In Gin, call `router.SetTrustedProxies(nil)` immediately after initializing the router to disable blindly trusting all proxies, ensuring `ClientIP()` resolves to the secure, actual network connection address.

## 2026-09-21 - Unhandled error when configuring trusted proxies
**Vulnerability:** The application configured Gin to not trust any proxies (`router.SetTrustedProxies(nil)`), but failed to handle the potential error returned by this function call. While `nil` is currently safe in the Gin framework version used, ignoring security configuration errors can lead to silently running in an insecure state if the framework changes or the configuration relies on external variables.
**Learning:** Security-critical configurations, such as setting trusted proxies to prevent IP spoofing, should strictly handle any returned errors to ensure the application fails to start rather than running insecurely.
**Prevention:** Always check and explicitly handle errors returned by security configuration methods in web frameworks, even if they are expected to succeed.

## 2026-09-21 - Insecure Directory Permissions for SQLite Database
**Vulnerability:** The database migration runner created the directory for the SQLite database with `0755` permissions. This allowed any user on the system to read the directory contents, potentially exposing the database file (and sensitive information within it) to unauthorized local users.
**Learning:** Directories containing sensitive files, like databases, must have restrictive permissions to prevent local information disclosure. Defaulting to `0755` for new directories is often too permissive for secure environments.
**Prevention:** Use `0750` or `0700` when creating directories that will hold sensitive files like databases, ensuring only the application owner and potentially the group can access them.

## 2026-09-21 - Integer Overflow in custom ParseUint
**Vulnerability:** A custom integer parsing function `ParseUint` contained an integer overflow vulnerability where the manual calculation `byte('0' + u - q*10)` could overflow before assigning to a byte slice if the order of operations wasn't strictly grouped to prevent intermediate unsigned underflow/overflow.
**Learning:** Custom implementations of core library functions (like `strconv.ParseUint` or `FormatUint`) are highly susceptible to integer overflow or bounds check failures.
**Prevention:** When optimizing parsing or formatting with manual byte loops, always use explicit parenthesis (e.g. `byte('0' + (u - q*10))`) to enforce safe evaluation order and prevent integer overflow regressions.
## 2026-09-22 - Suppress SAST False Positives for Hardcoded Test Credentials
**Vulnerability:** Static Application Security Testing (SAST) tools like `gosec` frequently flag dummy password hashes (e.g., bcrypt hashes) used in test fixture data as hardcoded credentials (rule G101).
**Learning:** While hardcoded credentials in production code are a critical vulnerability, dummy hashes in test files are false positives. Leaving them unaddressed can cause alert fatigue and obscure real vulnerabilities in automated security scans.
**Prevention:** Always use explicit suppression comments (e.g., `// #nosec G101`) to acknowledge and ignore false positives in test data, ensuring security scan results remain actionable and relevant.
## 2026-09-27 - Transaction Limits Bypass
**Vulnerability:** The application included a `TransactionLimitService` designed to enforce constraints on transaction amounts, velocity, and daily/monthly limits. However, this service was completely bypassed in the transaction handlers (`internal/handlers/transactions.go`) for `Deposit`, `Withdraw`, and `Transfer`, allowing attackers to bypass financial transfer restrictions.
**Learning:** Security controls implemented in separate service layers are ineffective if they are not explicitly wired into the core execution paths (handlers).
**Prevention:** Ensure that all intended security and business logic controls, such as rate limits and transaction limits, are actively invoked and enforced within the corresponding API handlers. Always verify that new security services are not just implemented but actually integrated.

## 2026-09-27 - Information Disclosure via /health/detailed Endpoint
**Vulnerability:** The `/health/detailed` endpoint in `internal/handlers/health.go` currently lacks authentication and exposes sensitive environment and database information, presenting a potential information disclosure vulnerability.
**Learning:** Endpoints providing detailed system, environment, or dependency information can aid attackers in reconnaissance and must be protected.
**Prevention:** Ensure that endpoints exposing detailed system health or configuration data require authentication and are restricted to administrative or monitoring roles.

## 2026-09-27 - Missing AutoMigrate for New Models in Tests
**Vulnerability:** When utilizing a new GORM model (like `TransactionLimit`) that wasn't previously queried in a specific test path, the tests failed with 'no such table' errors.
**Learning:** The in-memory test database relies on `AutoMigrate` in `internal/testutils/testdb_helper.go` to create schemas. Models must be explicitly added to this call to be available during testing.
**Prevention:** Always append new GORM models to the `AutoMigrate` function call in `testdb_helper.go` when they are introduced or utilized in new code paths to ensure test stability.

## 2026-10-01 - Standard Library Vulnerability Scan GO-2022-0535 Analysis
**Vulnerability:** GO-2022-0535 (CVE-2020-0601) is a legacy Windows x509 certificate validation bypass in the Go standard library (`crypto/x509`) fixed in Go versions 1.12.16 and 1.13.7. Automated vulnerability scanners flagged this historic finding against dependencies.
**Learning:** Modern Go application environments using Go 1.26.x runtimes are inherently patched against legacy standard library vulnerabilities like GO-2022-0535, as verified by `govulncheck` reporting zero symbol or package vulnerabilities.
**Prevention:** Ensure the application toolchain and `go.mod` declare a modern, fully patched Go version (e.g. Go 1.26.x+) and perform periodic `govulncheck` scans to confirm zero symbol or package exposure.

## 2026-10-16 - Null-Byte and Control Character Input Validation Bypass in Login Route
**Vulnerability:** The login handler (`LoginUser`) relied on JSON binding struct tags without validating username or password strings against control characters, line breaks (`\r`, `\n`), or null bytes (`\x00`). Null bytes in SQLite query parameters via CGO bindings truncate input strings, allowing attackers to bypass exact username matching (e.g. `user\x00injected` querying `user`), while null bytes in login passwords cause bcrypt truncation.
**Learning:** JSON binding tags enforce basic string length bounds, but fail to detect null bytes or control characters that alter underlying database C-string representations or password hashing libraries.
**Prevention:** Always perform explicit input validation with manual byte checks on authentication parameters in login endpoints to reject null bytes, carriage returns, and newlines before executing ORM/SQL queries or bcrypt comparisons.

## 2026-10-18 - Remediating Standard Library Vulnerability GO-2026-6609
**Vulnerability:** GO-2026-6609 flags potential excessive CPU consumption when parsing HTTP `Range` headers containing a large number of ranges in `net/http` static file functions (`FileServer`, `ServeContent`, `ServeFile`).
**Learning:** Standard library vulnerability findings reported by static dependency scanners require updating the Go runtime directive and toolchain in `go.mod`. Pure REST JSON APIs that do not invoke `net/http` file-serving functions do not execute range parsing, but declaring the patched Go toolchain version in `go.mod` ensures the application is built with the remediated standard library.
**Prevention:** Update `go.mod` with the explicit patched `toolchain` version (e.g., `toolchain go1.26.9`) to ensure builds use the patched standard library release.

## 2026-10-20 - Permanent Transaction Limit Denial-of-Service via Unreset Monthly Usage
**Vulnerability:** In `TransactionLimitService.resetUsageIfNeeded`, mutating `limit.LastResetDate` during the daily reset check caused the subsequent monthly reset check to evaluate against `now` instead of the original `LastResetDate`. As a result, `limit.MonthlyUsed` was never reset to 0 across month boundaries, leading to perpetual accumulation of monthly usage and eventual permanent denial-of-service on financial transactions for accounts.
**Learning:** Mutating state fields during sequential time-window checks invalidates subsequent conditions in the same function call.
**Prevention:** Always preserve initial state values in local variables before executing sequential condition checks that depend on those values.

## 2026-10-23 - TOCTOU Race Condition in Transaction Limit Recording
**Vulnerability:** `TransactionLimitService.RecordTransaction` previously calculated daily/monthly limit usage in memory and saved static values via `tx.Save(limit)`. Under concurrent requests, this caused a TOCTOU race condition where parallel transactions overwrote each other's limit usage, allowing users to bypass transaction limits.
**Learning:** Performing in-memory arithmetic on ORM struct instances followed by full struct persistence in high-concurrency environments causes lost updates.
**Prevention:** Use atomic database-level SQL expressions (`gorm.Expr("daily_used + ?", amount)`) via targeted column updates (`UpdateColumns`) to ensure concurrency-safe usage tracking without lost updates.

## 2026-10-24 - Uncontrolled Resource Consumption via Oversized Request Payloads
**Vulnerability:** The application lacked request body size restrictions on POST/PUT endpoints, allowing malicious clients to send arbitrarily large JSON payloads that cause server memory exhaustion and Denial of Service (CWE-400).
**Learning:** Default HTTP servers and Gin routers read request bodies without size caps unless explicitly constrained using `http.MaxBytesReader` or custom size limiting middleware.
**Prevention:** Always apply a request body size limiter middleware (e.g. `http.MaxBytesReader`) globally or to data-ingesting route groups to cap payload sizes before reading or parsing JSON bodies into memory.
## 2026-10-25 - Fix missing ReadHeaderTimeout for http.Server (CWE-400)
**Vulnerability:** The `http.Server` configuration in `cmd/server/main.go` lacked a `ReadHeaderTimeout`. This exposes the server to Slowloris attacks (CWE-400), where an attacker holds connections open by sending HTTP headers incredibly slowly, eventually exhausting the server's connection pool.
**Learning:** Go's default `http.Server` requires explicit configuration of `ReadHeaderTimeout` to protect against slowloris attacks. `ReadTimeout` alone is not always sufficient if headers are sent slowly but within the bounds of a larger overall timeout, or if it isn't set aggressively enough for just headers.
**Prevention:** Always configure `ReadHeaderTimeout` (typically a short value like 5 seconds) alongside other timeouts like `ReadTimeout`, `WriteTimeout`, and `IdleTimeout` when initializing an `http.Server` in Go.

## 2026-10-26 - Integer Underflow in JWT Subject Claim Conversion
**Vulnerability:** Negative or non-integer JWT subject claims (`sub <= 0` or `12.34`) were accepted by authentication middleware and cast directly to `uint`, causing integer underflow wrap-around (`uint(-1.0) == 18446744073709551615`) and allowing unauthorized access with invalid user IDs.
**Learning:** Converting untyped `float64` claims directly via `uint(subFloat)` without checking `v > 0`, non-finiteness, or `v == math.Trunc(v)` causes underflow wrap-around for negative numbers.
**Prevention:** Always validate bounds (`v > 0`), finiteness, and integer precision on JWT subject claims before converting to unsigned integer types and setting context keys.

## 2026-10-29 - CRLF Log Injection via Unsanitized User-Agent Header in Audit Logs
**Vulnerability:** Audit log entries recorded raw `User-Agent` HTTP request headers directly into database and application audit logs without character sanitization or length restrictions, exposing structured log consumers to CRLF log injection (CWE-117) and database storage bloat.
**Learning:** Request headers supplied directly by clients cannot be trusted in log entries or storage, even in background audit services.
**Prevention:** Sanitize incoming client HTTP headers using `validation.SanitizeUserAgent` to truncate strings to max length (e.g. 512 bytes) and convert ASCII control characters (`< 32` or `127`, including `\r`, `\n`, `\x00`) to spaces before recording audit logs.

## 2026-10-29 - Standard Library Vulnerability GO-2026-6809 in crypto/x509
**Vulnerability:** GO-2026-6809 flags compromised intermediate x509 certificates that were not distrusted by older Go standard library `crypto/x509` verification functions.
**Learning:** Standard library vulnerability findings flagged against stdlib components in Go require upgrading the Go directive and runtime/toolchain version in `go.mod` to ensure the application builds against patched `crypto/x509` implementations.
**Prevention:** Maintain `go.mod` up to date with the latest Go patch version (e.g., `go 1.26.10`) to inherit standard library security fixes.
