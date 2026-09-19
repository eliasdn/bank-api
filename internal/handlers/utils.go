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
