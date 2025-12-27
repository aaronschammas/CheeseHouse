package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger middleware para logging detallado de requests
func RequestLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Formato personalizado de logs
		var statusColor, methodColor, resetColor string

		// Colores según el status code
		if param.IsOutputColor() {
			statusColor = param.StatusCodeColor()
			methodColor = param.MethodColor()
			resetColor = param.ResetColor()
		}

		// Emoji según el método
		var methodEmoji string
		switch param.Method {
		case "GET":
			methodEmoji = "📥"
		case "POST":
			methodEmoji = "📤"
		case "PUT":
			methodEmoji = "✏️"
		case "DELETE":
			methodEmoji = "🗑️"
		case "PATCH":
			methodEmoji = "🔧"
		default:
			methodEmoji = "📋"
		}

		// Emoji según status code
		var statusEmoji string
		switch {
		case param.StatusCode >= 200 && param.StatusCode < 300:
			statusEmoji = "✅"
		case param.StatusCode >= 300 && param.StatusCode < 400:
			statusEmoji = "↩️"
		case param.StatusCode >= 400 && param.StatusCode < 500:
			statusEmoji = "⚠️"
		case param.StatusCode >= 500:
			statusEmoji = "❌"
		}

		return fmt.Sprintf("%s %s[%s]%s %s %3d %s| %13v | %15s | %s%-7s%s %s %#v %s\n%s",
			statusEmoji,
			statusColor,
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			resetColor,
			methodEmoji,
			param.StatusCode,
			statusColor, param.Latency, resetColor,
			param.ClientIP,
			methodColor, param.Method, resetColor,
			param.Path,
			param.ErrorMessage,
		)
	})
}

// APILogger middleware específico para APIs con más detalles
func APILogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Tiempo de inicio
		startTime := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Procesar request
		c.Next()

		// Calcular latencia
		latency := time.Since(startTime)

		// Información del usuario si está autenticado
		userInfo := "Anonymous"
		if email, exists := c.Get("user_email"); exists {
			userInfo = fmt.Sprintf("%v", email)
		}

		// Log detallado
		fmt.Printf("🔍 API Request | "+
			"Time: %s | "+
			"Status: %d | "+
			"Latency: %v | "+
			"IP: %s | "+
			"Method: %s | "+
			"Path: %s | "+
			"Query: %s | "+
			"User: %s | "+
			"Errors: %s\n",
			startTime.Format("15:04:05"),
			c.Writer.Status(),
			latency,
			c.ClientIP(),
			c.Request.Method,
			path,
			query,
			userInfo,
			c.Errors.ByType(gin.ErrorTypePrivate).String(),
		)
	}
}

// ErrorLogger middleware para capturar y loggear errores
func ErrorLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Si hay errores, logearlos
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				fmt.Printf("❌ ERROR | "+
					"Time: %s | "+
					"Path: %s | "+
					"IP: %s | "+
					"Type: %s | "+
					"Error: %v\n",
					time.Now().Format("2006/01/02 15:04:05"),
					c.Request.URL.Path,
					c.ClientIP(),
					err.Type,
					err.Err,
				)
			}
		}
	}
}

// SecurityLogger middleware para eventos de seguridad
func SecurityLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Loguear intentos de acceso no autorizados
		if c.Writer.Status() == 401 || c.Writer.Status() == 403 {
			fmt.Printf("🔒 SECURITY | "+
				"Time: %s | "+
				"Status: %d | "+
				"IP: %s | "+
				"Method: %s | "+
				"Path: %s | "+
				"UserAgent: %s\n",
				time.Now().Format("2006/01/02 15:04:05"),
				c.Writer.Status(),
				c.ClientIP(),
				c.Request.Method,
				c.Request.URL.Path,
				c.Request.UserAgent(),
			)
		}
	}
}

// PerformanceLogger middleware para monitorear performance
func PerformanceLogger(slowThreshold time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		c.Next()

		latency := time.Since(startTime)

		// Loguear solo requests lentos
		if latency > slowThreshold {
			fmt.Printf("⚡ SLOW REQUEST | "+
				"Time: %s | "+
				"Latency: %v | "+
				"Threshold: %v | "+
				"Path: %s | "+
				"Method: %s\n",
				startTime.Format("15:04:05"),
				latency,
				slowThreshold,
				c.Request.URL.Path,
				c.Request.Method,
			)
		}
	}
}
