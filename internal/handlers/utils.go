package handlers

import (
	"bank-api/internal/validation"
	"github.com/gin-gonic/gin"
)

// getUserID safely extracts the user ID from the gin Context.
// It handles uint, float64 (common from JSON/JWT), and string representations
// to prevent silent type assertion failures and rate-limiting bypasses.
func getUserID(c *gin.Context) uint {
	val, exists := c.Get("userID")
	if !exists {
		return 0
	}

	switch v := val.(type) {
	case uint:
		return v
	case float64:
		return uint(v)
	case string:
		if id, err := validation.ParseUint(v, 0); err == nil {
			return uint(id)
		}
	}

	return 0
}

// unauthorizedErrorJSON is a pre-allocated JSON response used to avoid
// map allocations (gin.H) and JSON reflection overhead in hot paths
// for unauthenticated requests.
var unauthorizedErrorJSON = []byte(`{"error":"unauthorized"}`)

var (
	atLeastOneFieldMustBeProvidedErrorJSON = []byte(`{"error":"at least one field must be provided"}`)
	databaseErrorJSON                      = []byte(`{"error":"database error"}`)
	emailAlreadyInUseErrorJSON             = []byte(`{"error":"email already in use"}`)
	failedToCreateUserErrorJSON            = []byte(`{"error":"failed to create user"}`)
	failedToDeleteUserAccountsErrorJSON    = []byte(`{"error":"failed to delete user accounts"}`)
	failedToDeleteUserErrorJSON            = []byte(`{"error":"failed to delete user"}`)
	failedToGenerateTokenErrorJSON         = []byte(`{"error":"failed to generate token"}`)
	failedToHashPasswordErrorJSON          = []byte(`{"error":"failed to hash password"}`)
	failedToUpdateUserErrorJSON            = []byte(`{"error":"failed to update user"}`)
	invalidCredentialsErrorJSON            = []byte(`{"error":"invalid credentials"}`)
	invalidEmailFormatErrorJSON            = []byte(`{"error":"invalid email format"}`)
	invalidUserIDErrorJSON                 = []byte(`{"error":"invalid user ID"}`)
	userNotFoundErrorJSON                  = []byte(`{"error":"user not found"}`)
	usernameOrEmailAlreadyExistsErrorJSON  = []byte(`{"error":"username or email already exists"}`)

	invalidRequestErrorJSON           = []byte(`{"error":"invalid request"}`)
	invalidAccountIDErrorJSON         = []byte(`{"error":"invalid account ID"}`)
	accountNotFoundErrorJSON          = []byte(`{"error":"account not found"}`)
	failedToRetrieveAccountsErrorJSON = []byte(`{"error":"failed to retrieve accounts"}`)
	failedToCreateAccountErrorJSON    = []byte(`{"error":"failed to create account"}`)
	failedToRetrieveAccountErrorJSON  = []byte(`{"error":"failed to retrieve account"}`)

	invalidDepositAmountErrorJSON              = []byte(`{"error":"invalid deposit amount"}`)
	invalidWithdrawalAmountErrorJSON           = []byte(`{"error":"invalid withdrawal amount"}`)
	insufficientFundsErrorJSON                 = []byte(`{"error":"insufficient funds"}`)
	failedToUpdateBalanceErrorJSON             = []byte(`{"error":"failed to update balance"}`)
	failedToReloadAccountErrorJSON             = []byte(`{"error":"failed to reload account"}`)
	failedToRecordTransactionErrorJSON         = []byte(`{"error":"failed to record transaction"}`)
	transactionFailedErrorJSON                 = []byte(`{"error":"transaction failed"}`)
	sourceAccountNotFoundErrorJSON             = []byte(`{"error":"source account not found"}`)
	invalidTransferRequestErrorJSON            = []byte(`{"error":"invalid transfer request"}`)
	destinationAccountNotFoundErrorJSON        = []byte(`{"error":"destination account not found"}`)
	failedToUpdateSourceAccountErrorJSON       = []byte(`{"error":"failed to update source account"}`)
	failedToUpdateDestinationAccountErrorJSON  = []byte(`{"error":"failed to update destination account"}`)
	failedToReloadSourceAccountErrorJSON       = []byte(`{"error":"failed to reload source account"}`)
	failedToReloadDestinationAccountErrorJSON  = []byte(`{"error":"failed to reload destination account"}`)
	failedToRecordOutgoingTransactionErrorJSON = []byte(`{"error":"failed to record outgoing transaction"}`)
	failedToRecordIncomingTransactionErrorJSON = []byte(`{"error":"failed to record incoming transaction"}`)
	transferFailedErrorJSON                    = []byte(`{"error":"transfer failed"}`)
	failedToRetrieveTransactionsErrorJSON      = []byte(`{"error":"failed to retrieve transactions"}`)
	userUpdatedSuccessfullyJSON                = []byte(`{"message":"User updated successfully"}`)
	userDeletedSuccessfullyJSON                = []byte(`{"message":"User deleted successfully"}`)
)

// errorResponse provides a zero-allocation schema for dynamic error messages.
type errorResponse struct {
	Error string `json:"error"`
}
