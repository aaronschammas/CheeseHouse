package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"CheeseHouse/internal/services"
)

// AuthMiddleware middleware para autenticación JWT
type AuthMiddleware struct {
	authService *services.AuthService
}

// NewAuthMiddleware crea una nueva instancia del middleware de autenticación
func NewAuthMiddleware(authService *services.AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

// RequireAuth middleware que requiere autenticación
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Obtener token del header Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// Si no hay header, buscar en cookie
			token, err := c.Cookie("auth_token")
			if err != nil || token == "" {
				log.Printf("🔒 Acceso denegado: No hay token - IP: %s, Path: %s", c.ClientIP(), c.Request.URL.Path)
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "No autorizado",
					"message": "Token de autenticación requerido",
				})
				c.Abort()
				return
			}
			authHeader = "Bearer " + token
		}

		// Extraer token del header "Bearer <token>"
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			log.Printf("🔒 Acceso denegado: Formato de token inválido - IP: %s", c.ClientIP())
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "No autorizado",
				"message": "Formato de token inválido",
			})
			c.Abort()
			return
		}

		// Validar token
		claims, err := m.authService.ValidateToken(tokenString)
		if err != nil {
			log.Printf("🔒 Acceso denegado: Token inválido - %v - IP: %s", err, c.ClientIP())
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "No autorizado",
				"message": "Token inválido o expirado",
			})
			c.Abort()
			return
		}

		// Obtener usuario completo
		usuario, err := m.authService.GetUsuarioFromToken(tokenString)
		if err != nil {
			log.Printf("🔒 Acceso denegado: Usuario no encontrado - %v - IP: %s", err, c.ClientIP())
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "No autorizado",
				"message": "Usuario no válido",
			})
			c.Abort()
			return
		}

		// Guardar información del usuario en el contexto
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_name", claims.Nombre)
		c.Set("rol_id", claims.RolID)
		c.Set("rol_name", claims.RolName)
		c.Set("usuario", usuario)

		log.Printf("✅ Usuario autenticado: %s (%s) - Path: %s", claims.Email, claims.RolName, c.Request.URL.Path)

		c.Next()
	}
}

// RequireAdmin middleware que requiere rol de administrador
func (m *AuthMiddleware) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Primero verificar autenticación
		m.RequireAuth()(c)

		if c.IsAborted() {
			return
		}

		// Verificar que sea admin
		rolName, exists := c.Get("rol_name")
		if !exists || rolName != "admin" {
			log.Printf("🔒 Acceso denegado: Se requiere rol admin - Usuario: %v, Rol: %v",
				c.GetString("user_email"), rolName)
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Acceso denegado",
				"message": "Se requieren permisos de administrador",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// OptionalAuth middleware que permite autenticación opcional
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			token, err := c.Cookie("auth_token")
			if err != nil || token == "" {
				// No hay token, continuar sin autenticación
				c.Next()
				return
			}
			authHeader = "Bearer " + token
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.Next()
			return
		}

		// Intentar validar token
		claims, err := m.authService.ValidateToken(tokenString)
		if err == nil {
			usuario, err := m.authService.GetUsuarioFromToken(tokenString)
			if err == nil {
				c.Set("user_id", claims.UserID)
				c.Set("user_email", claims.Email)
				c.Set("user_name", claims.Nombre)
				c.Set("rol_id", claims.RolID)
				c.Set("rol_name", claims.RolName)
				c.Set("usuario", usuario)
				c.Set("authenticated", true)
			}
		}

		c.Next()
	}
}

// GetUserID helper para obtener el ID del usuario del contexto
func GetUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	id, ok := userID.(uint)
	return id, ok
}

// GetUserEmail helper para obtener el email del usuario del contexto
func GetUserEmail(c *gin.Context) (string, bool) {
	email, exists := c.Get("user_email")
	if !exists {
		return "", false
	}
	emailStr, ok := email.(string)
	return emailStr, ok
}

// IsAdmin helper para verificar si el usuario es admin
func IsAdmin(c *gin.Context) bool {
	rolName, exists := c.Get("rol_name")
	if !exists {
		return false
	}
	return rolName == "admin"
}

// IsAuthenticated helper para verificar si hay un usuario autenticado
func IsAuthenticated(c *gin.Context) bool {
	_, exists := c.Get("user_id")
	return exists
}
