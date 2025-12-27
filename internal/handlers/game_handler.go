package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"CheeseHouse/internal/models"
	"CheeseHouse/internal/services"
)

// GameHandler maneja todas las rutas relacionadas con el juego
type GameHandler struct {
	gameService *services.GameService
}

// NewGameHandler crea una nueva instancia del handler del juego
func NewGameHandler(gameService *services.GameService) *GameHandler {
	return &GameHandler{
		gameService: gameService,
	}
}

// ShowGame muestra la página principal del juego
func (h *GameHandler) ShowGame(c *gin.Context) {
	// Obtener configuración del juego para el template
	gameConfig := h.gameService.GetConfiguracionJuego()

	// Datos para el template
	data := gin.H{
		"titulo":         "CheeseHouse - Juego de Timing",
		"restaurante":    gameConfig["restaurante"],
		"tolerancia":     gameConfig["tolerancia"],
		"tiempo_min":     gameConfig["tiempo_min"],
		"tiempo_max":     gameConfig["tiempo_max"],
		"descuento_win":  gameConfig["descuento_ganador"],
		"descuento_lose": gameConfig["descuento_perdedor"],
	}

	c.HTML(http.StatusOK, "game.html", data)
}

// SubmitGameResult procesa el resultado del juego enviado por AJAX
func (h *GameHandler) SubmitGameResult(c *gin.Context) {
	var gameResult models.GameResult

	// Parsear JSON del request
	if err := c.ShouldBindJSON(&gameResult); err != nil {
		log.Printf("❌ Error parsing game result: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Datos del juego inválidos",
			"error":   err.Error(),
		})
		return
	}

	// Log del intento de juego
	log.Printf("🎮 Juego recibido: %s %s (%s) - Objetivo: %.1fs, Obtenido: %.2fs",
		gameResult.ClienteData.Nombre,
		gameResult.ClienteData.Apellido,
		gameResult.ClienteData.Telefono,
		gameResult.Resultado.TiempoObjetivo,
		gameResult.Resultado.TiempoObtenido)

	// Procesar resultado con el servicio
	response, err := h.gameService.ProcesarResultadoJuego(gameResult)
	if err != nil {
		log.Printf("❌ Error procesando juego: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error interno del servidor",
			"error":   err.Error(),
		})
		return
	}

	// Log del resultado
	if response.Success {
		log.Printf("✅ Juego procesado: Código %s (%d%% descuento) para %s",
			response.Codigo, response.Descuento, gameResult.ClienteData.Telefono)
	} else {
		log.Printf("⚠️  Juego rechazado: %s", response.Message)
	}

	// Retornar respuesta JSON
	c.JSON(http.StatusOK, response)
}

// GetGameStats obtiene estadísticas públicas del juego
func (h *GameHandler) GetGameStats(c *gin.Context) {
	stats, err := h.gameService.GetEstadisticasGenerales()
	if err != nil {
		log.Printf("❌ Error obteniendo estadísticas: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error obteniendo estadísticas",
		})
		return
	}

	// Estadísticas públicas (sin datos sensibles)
	publicStats := gin.H{
		"total_clientes":       stats.TotalClientes,
		"total_partidas":       stats.TotalPartidas,
		"porcentaje_victorias": stats.PorcentajeVictorias,
		"jugaron_hoy":          stats.JugaronHoy,
		"restaurante":          "CheeseHouse",
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"estadisticas": publicStats,
	})
}

// GetClientByPhone obtiene información básica de un cliente por teléfono
func (h *GameHandler) GetClientByPhone(c *gin.Context) {
	telefono := c.Param("phone")

	if telefono == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Teléfono requerido",
		})
		return
	}

	cliente, err := h.gameService.GetClientePorTelefono(telefono)
	if err != nil {
		// Cliente no encontrado no es error crítico
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Cliente no encontrado",
		})
		return
	}

	// Información básica del cliente (sin datos sensibles)
	clientePublic := gin.H{
		"nombre":         cliente.Nombre,
		"apellido":       cliente.Apellido,
		"total_juegos":   cliente.TotalJuegos,
		"juegos_ganados": cliente.JuegosGanados,
		"tipo_cliente":   cliente.TipoCliente,
		"ultimo_juego":   cliente.FechaUltimoJuego,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"cliente": clientePublic,
	})
}

// GenerateTargetTime genera un nuevo tiempo objetivo (para el frontend)
func (h *GameHandler) GenerateTargetTime(c *gin.Context) {
	targetTime := h.gameService.GenerarTiempoObjetivo()

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"target_time": targetTime,
	})
}

// GetGameConfig obtiene la configuración del juego para el frontend
func (h *GameHandler) GetGameConfig(c *gin.Context) {
	config := h.gameService.GetConfiguracionJuego()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"config":  config,
	})
}

// Health endpoint para verificar el estado del servicio de juego
func (h *GameHandler) Health(c *gin.Context) {
	// Verificar que el servicio esté funcionando
	stats, err := h.gameService.GetEstadisticasGenerales()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "Servicio de juego no disponible",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         "ok",
		"service":        "CheeseHouse Game Service",
		"total_clientes": stats.TotalClientes,
		"total_partidas": stats.TotalPartidas,
	})
}

// TestGame endpoint para probar el sistema (solo en desarrollo)
func (h *GameHandler) TestGame(c *gin.Context) {
	// Solo disponible en modo desarrollo
	if gin.Mode() == gin.ReleaseMode {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Endpoint no disponible en producción",
		})
		return
	}

	// Simular un juego de prueba
	testResult := models.GameResult{
		ClienteData: models.ClienteData{
			Nombre:   "Test",
			Apellido: "Usuario",
			Telefono: "+5491123456789",
		},
		Resultado: models.Resultado{
			Gano:           true,
			TiempoObjetivo: 7.5,
			TiempoObtenido: 7.3,
		},
	}

	response, err := h.gameService.ProcesarResultadoJuego(testResult)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Juego de prueba ejecutado",
		"result":  response,
	})
}

// Middleware para logging de requests de juego
func (h *GameHandler) GameLoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("🎮 %s - [%s] \"%s %s %s %d %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format("15:04:05"),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
		)
	})
}
