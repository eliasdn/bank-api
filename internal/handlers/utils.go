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
)
