package middleware

import (
	"strings"

	"github.com/aouiniamine/aoui-drive/internal/features/auth/dto"
	"github.com/aouiniamine/aoui-drive/internal/features/auth/service"
	"github.com/aouiniamine/aoui-drive/pkg/response"
	"github.com/labstack/echo/v4"
)

const ClientIDKey = "client_id"

func Auth(authService service.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return response.Unauthorized(c, "missing authorization header")
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return response.Unauthorized(c, "invalid authorization header format")
			}

			claims, err := authService.ValidateToken(parts[1])
			if err != nil {
				return response.Unauthorized(c, "invalid or expired token")
			}

			c.Set(ClientIDKey, claims.ClientID)
			return next(c)
		}
	}
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
