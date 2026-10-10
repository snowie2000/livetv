package handler

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// AuthRequired ensures that only authenticated session users can access the endpoint
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if sessions.Default(c).Get("logined") != true {
			c.String(http.StatusUnauthorized, "Unauthorized")
			c.Abort()
			return
		}
		c.Next()
	}
}

// IsSecure checks if the request is over HTTPS (direct TLS or reverse-proxy forwarded)
func IsSecure(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	if c.GetHeader("X-Forwarded-Proto") == "https" || c.GetHeader("x-forwarded-proto") == "https" {
		return true
	}
	if c.GetHeader("X-Forwarded-Ssl") == "on" || c.GetHeader("x-forwarded-ssl") == "on" {
		return true
	}
	return false
}

// GetSessionOptions returns session cookie options compatible with both HTTP and HTTPS reverse proxies
func GetSessionOptions(c *gin.Context) sessions.Options {
	return sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   IsSecure(c),
	}
}
