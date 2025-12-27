package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Environment    string
	RestaurantName string
	Location       string

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// WhatsApp
	WhatsAppToken string
	WhatsAppURL   string

	// JWT
	JWTSecret string

	// Game
	MinTargetTime float64
	MaxTargetTime float64
	WinDiscount   int
	LoseDiscount  int
	Tolerance     float64
}

func Load() *Config {
	cfg := &Config{
		Environment:    getEnv("ENV", "development"),
		RestaurantName: getEnv("RESTAURANT_NAME", "CheeseHouse"),
		Location:       getEnv("LOCATION", "Centro"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "root"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "cheesehouse"),

		WhatsAppToken: getEnv("WHATSAPP_TOKEN", ""),
		WhatsAppURL:   getEnv("WHATSAPP_URL", "https://api.twilio.com"),

		JWTSecret: getEnv("JWT_SECRET", "your-secret-key"),

		MinTargetTime: 5.0,
		MaxTargetTime: 20.0,
		WinDiscount:   30,
		LoseDiscount:  10,
		Tolerance:     0.1,
	}

	// Override game config from env if present
	if val := getEnv("MIN_TARGET_TIME", ""); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			cfg.MinTargetTime = f
		}
	}
	if val := getEnv("MAX_TARGET_TIME", ""); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			cfg.MaxTargetTime = f
		}
	}
	if val := getEnv("WIN_DISCOUNT", ""); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			cfg.WinDiscount = i
		}
	}
	if val := getEnv("LOSE_DISCOUNT", ""); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			cfg.LoseDiscount = i
		}
	}
	if val := getEnv("TOLERANCE", ""); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			cfg.Tolerance = f
		}
	}

	return cfg
}

func (c *Config) Validate() []string {
	var errors []string

	if c.DBHost == "" {
		errors = append(errors, "DB_HOST is required")
	}
	if c.DBUser == "" {
		errors = append(errors, "DB_USER is required")
	}
	if c.DBName == "" {
		errors = append(errors, "DB_NAME is required")
	}
	if c.WhatsAppToken == "" {
		errors = append(errors, "WHATSAPP_TOKEN is required for production")
	}
	if c.JWTSecret == "" {
		errors = append(errors, "JWT_SECRET is required")
	}

	return errors
}

func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

func (c *Config) LogConfig() {
	fmt.Println("🧀 Configuration loaded:")
	fmt.Printf("   Environment: %s\n", c.Environment)
	fmt.Printf("   Restaurant: %s (%s)\n", c.RestaurantName, c.Location)
	fmt.Printf("   Database: %s@%s:%s/%s\n", c.DBUser, c.DBHost, c.DBPort, c.DBName)
	fmt.Printf("   Game: %.1f-%.1fs, Win:%d%%, Lose:%d%%, Tol:%.1f\n",
		c.MinTargetTime, c.MaxTargetTime, c.WinDiscount, c.LoseDiscount, c.Tolerance)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
