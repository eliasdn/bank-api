package middleware

import (
	"bank-api/internal/validation"
	"net/http"
	"strings"

	"bank-api/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AuthMiddleware struct {
	config *config.AppConfig
}

func NewAuthMiddleware(cfg *config.AppConfig) *AuthMiddleware {
	return &AuthMiddleware{config: cfg}
}

func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Bearer token not found"})
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(m.config.JWT.Secret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}

		// RFC 7519 defines 'sub' as a string, but many implementations use numbers.
		// Accommodate float64 and numeric strings (converting to uint for DB lookups),
		// while preserving non-numeric strings for external conventions.
		if subFloat, ok := claims["sub"].(float64); ok {
			c.Set("userID", uint(subFloat))
		} else if subStr, ok := claims["sub"].(string); ok {
			if id, err := validation.ParseUint(subStr, 0); err == nil {
				c.Set("userID", uint(id))
			} else {
				c.Set("userID", subStr)
			}
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
			return
		}

		c.Next()
	}
}
