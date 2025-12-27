package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"CheeseHouse/internal/config"
	"CheeseHouse/internal/database"
	"CheeseHouse/internal/handlers"
	"CheeseHouse/internal/repository"
	"CheeseHouse/internal/services"
)

func main() {
	// Cargar variables de entorno
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No se encontró archivo .env, usando variables del sistema")
	}

	// Inicializar configuración
	cfg := config.Load()
	cfg.LogConfig()

	// Validar configuración
	if errors := cfg.Validate(); len(errors) > 0 {
		log.Println("⚠️  Advertencias de configuración:")
		for _, err := range errors {
			log.Printf("   - %s", err)
		}
	}

	// Conectar a la base de datos
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal("❌ Error fatal conectando a la base de datos:", err)
	}

	// Inicializar repositorios
	clienteRepo := repository.NewClienteRepository(db.DB)
	voucherRepo := repository.NewVoucherRepository(db.DB)

	// Inicializar servicios
	whatsappService := services.NewWhatsAppService(cfg)
	gameService := services.NewGameService(cfg, clienteRepo, voucherRepo, whatsappService)

	// Inicializar handlers
	gameHandler := handlers.NewGameHandler(gameService)

	// Configurar router
	router := setupRouter(gameHandler, db, cfg, whatsappService)

	// Iniciar servidor
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println(" ================================")
	log.Println("     CHEESEHOUSE TIMING        ")
	log.Println(" ================================")
	log.Printf(" Servidor iniciando en puerto %s", port)
	log.Printf("Juego disponible en: http://localhost:%s", port)
	log.Printf(" Health check: http://localhost:%s/health", port)
	log.Println(" ================================")

	if err := router.Run(":" + port); err != nil {
		log.Fatal(" Error fatal iniciando servidor:", err)
	}
}

func setupRouter(
	gameHandler *handlers.GameHandler,
	db *database.Database,
	cfg *config.Config,
	whatsappService *services.WhatsAppService,
) *gin.Engine {
	// Modo release en producción
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Middleware de CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Middleware de logging personalizado
	router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("🧀 %s - [%s] \"%s %s %s %d %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format("15:04:05"),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
		)
	}))

	// Middleware de recovery
	router.Use(gin.Recovery())

	// Servir archivos estáticos
	router.Static("/static", "./web/static")

	// Cargar templates HTML
	router.LoadHTMLGlob("web/templates/*")

	// ===============================
	// RUtAS PARA EL JUEGOVICH
	// ===============================

	// Página principal del juego
	router.GET("/", gameHandler.ShowGame)

	// API del juego
	gameAPI := router.Group("/api/game")
	{
		gameAPI.POST("/submit", gameHandler.SubmitGameResult)
		gameAPI.GET("/stats", gameHandler.GetGameStats)
		gameAPI.GET("/config", gameHandler.GetGameConfig)
		gameAPI.GET("/target", gameHandler.GenerateTargetTime)

		// Solo en desarrollo
		if !cfg.IsProduction() {
			gameAPI.POST("/test", gameHandler.TestGame)
		}
	}

	// API de clientes (consultas públicas limitadas)
	clientsAPI := router.Group("/api/clients")
	{
		clientsAPI.GET("/:phone", gameHandler.GetClientByPhone)
	}

	// ===============================
	// HEALTH CHECKS
	// ===============================

	router.GET("/health", func(c *gin.Context) {
		// Verificar salud de la base de datos
		dbHealth := "ok"
		if err := db.Health(); err != nil {
			dbHealth = "error: " + err.Error()
		}

		// Verificar salud del servicio de juego
		gameHealth := "ok"

		// Verificar estado de WhatsApp
		whatsappStatus := whatsappService.GetStatus()

		status := 200
		if dbHealth != "ok" || gameHealth != "ok" {
			status = 503
		}

		c.JSON(status, gin.H{
			"status":       "running",
			"service":      "CheeseHouse Timing Game",
			"version":      "1.0.0",
			"environment":  cfg.Environment,
			"database":     dbHealth,
			"game_service": gameHealth,
			"whatsapp":     whatsappStatus,
			"db_stats":     db.GetStats(),
		})
	})

	// Endpoint para información del sistema
	router.GET("/info", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"restaurante": cfg.RestaurantName,
			"ubicacion":   cfg.Location,
			"version":     "1.0.0",
			"endpoints": map[string]string{
				"juego":      "/",
				"api_submit": "/api/game/submit",
				"api_stats":  "/api/game/stats",
				"health":     "/health",
			},
		})
	})

	// 404 Handler
	router.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{
			"error":   "Endpoint no encontrado",
			"message": "La ruta solicitada no existe",
			"path":    c.Request.URL.Path,
		})
	})

	return router
}
