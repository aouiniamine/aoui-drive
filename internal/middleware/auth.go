package middleware

import (
	"net/http"
	"strings"

	"github.com/aouiniamine/aoui-drive/internal/features/auth/dto"
	"github.com/aouiniamine/aoui-drive/internal/features/auth/service"
	"github.com/aouiniamine/aoui-drive/pkg/response"
	"github.com/labstack/echo/v4"
)

const (
	ClientIDKey       = "client_id"
	SessionCookieName = "session"
)

// Auth middleware checks for Bearer token first, then falls back to session cookie.
// For UI routes (starting with /ui), it redirects to login on failure.
// For API routes, it returns JSON error responses.
func Auth(authService service.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var token string

			// First, try Bearer token from Authorization header
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
					token = parts[1]
				}
			}

			// If no Bearer token, try session cookie
			if token == "" {
				cookie, cookieErr := c.Cookie(SessionCookieName)
				if cookieErr == nil && cookie.Value != "" {
					token = cookie.Value
				}
			}

			// No token found
			if token == "" {
				return authError(c, "missing authorization")
			}

			// Validate token
			claims, err := authService.ValidateToken(token)
			if err != nil {
				// Clear invalid cookie if present
				clearSessionCookie(c)
				return authError(c, "invalid or expired token")
			}

			c.Set(ClientIDKey, claims.ClientID)
			return next(c)
		}
	}
}

// authError returns appropriate error response based on request path
func authError(c echo.Context, message string) error {
	path := c.Request().URL.Path
	if strings.HasPrefix(path, "/ui") {
		return c.Redirect(http.StatusFound, "/ui/login?error="+message)
	}
	return response.Unauthorized(c, message)
}

// clearSessionCookie removes the session cookie
func clearSessionCookie(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

func RequireAdmin(authService service.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			clientID, ok := c.Get(ClientIDKey).(string)
			if !ok || clientID == "" {
				return response.Unauthorized(c, "unauthorized")
			}

			client, err := authService.GetClientByID(c.Request().Context(), clientID)
			if err != nil {
				return response.Unauthorized(c, "unauthorized")
			}

			if dto.Role(client.Role) != dto.RoleAdmin {
				return response.Forbidden(c, "admin access required")
			}

			return next(c)
		}
	}
}

func GetClientID(c echo.Context) string {
	clientID, _ := c.Get(ClientIDKey).(string)
	return clientID
}
