package middleware

import (
	"strings"

	"daily-notes-api/pkg/auth"
	"daily-notes-api/pkg/response"

	"github.com/gin-gonic/gin"
)

const (
	AuthUserKey = "auth_user"
	UserIDKey   = "user_id"
)

func AuthMiddleware(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "Authorization header required")
			c.Abort()
			return
		}

		// Expected format: "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c, "Invalid authorization header format")
			c.Abort()
			return
		}

		token := parts[1]
		claims, err := jwtManager.ValidateToken(token)
		if err != nil {
			response.Unauthorized(c, "Invalid or expired token")
			c.Abort()
			return
		}

		// Store user info in context
		c.Set(AuthUserKey, claims)
		c.Set(UserIDKey, claims.UserID)
		c.Next()
	}
}

// GetCurrentUser helper function to get current user from context
func GetCurrentUser(c *gin.Context) (*auth.JWTClaims, bool) {
	user, exists := c.Get(AuthUserKey)
	if !exists {
		return nil, false
	}

	claims, ok := user.(*auth.JWTClaims)
	return claims, ok
}

// GetCurrentUserID helper function to get current user ID from context
func GetCurrentUserID(c *gin.Context) (int, bool) {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return 0, false
	}

	id, ok := userID.(int)
	return id, ok
}
