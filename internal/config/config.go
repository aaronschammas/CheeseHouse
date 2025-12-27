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
	WhatsAppToken         string
	WhatsAppURL           string
	WhatsAppPhoneNumberID string

	// JWT
	JWTSecret string

	// Game
	Game GameConfig
}

type PhoneValidation struct {
	CountryCode string
	MinLength   int
	MaxLength   int
	AllowIntl   bool
	AreaCodes   []string
}

type GameConfig struct {
	MinTargetTime        float64
	MaxTargetTime        float64
	WinDiscount          int
	LoseDiscount         int
	Tolerance            float64
	VoucherValidityDays  int
	GamesRequireApproval int
}

func Load() *Config {
	cfg := &Config{
		Environment:    getEnv("ENV", "development"),
		RestaurantName: getEnv("RESTAURANT_NAME", "CheeseHouse"),
		Location:       getEnv("LOCATION", "Centro"),

		DBHost:     getEnv("DB_HOST", "127.0.0.1"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "root"),
		DBPassword: getEnv("DB_PASSWORD", "12345"),
		DBName:     getEnv("DB_NAME", "cheesehouse"),

		WhatsAppToken:         getEnv("WHATSAPP_TOKEN", ""),
		WhatsAppURL:           getEnv("WHATSAPP_URL", "https://api.twilio.com"),
		WhatsAppPhoneNumberID: getEnv("WHATSAPP_PHONE_NUMBER_ID", ""),

		JWTSecret: getEnv("JWT_SECRET", "your-secret-key"),

		Game: GameConfig{
			MinTargetTime:        5.0,
			MaxTargetTime:        20.0,
			WinDiscount:          30,
			LoseDiscount:         10,
			Tolerance:            0.1,
			VoucherValidityDays:  30,
			GamesRequireApproval: 3,
		},
	}

	// Override game config from env if present
	if val := getEnv("MIN_TARGET_TIME", ""); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			cfg.Game.MinTargetTime = f
		}
	}
	if val := getEnv("MAX_TARGET_TIME", ""); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			cfg.Game.MaxTargetTime = f
		}
	}
	if val := getEnv("WIN_DISCOUNT", ""); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			cfg.Game.WinDiscount = i
		}
	}
	if val := getEnv("LOSE_DISCOUNT", ""); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			cfg.Game.LoseDiscount = i
		}
	}
	if val := getEnv("TOLERANCE", ""); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			cfg.Game.Tolerance = f
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
		c.Game.MinTargetTime, c.Game.MaxTargetTime, c.Game.WinDiscount, c.Game.LoseDiscount, c.Game.Tolerance)
}

func (c *Config) GetWhatsAppTemplates() map[string]string {
	return map[string]string{
		"voucher_ganador":  "voucher_ganador",
		"voucher_perdedor": "voucher_perdedor",
		"bienvenida":       "bienvenida",
		"recordatorio":     "recordatorio",
	}
}

func (c *Config) GetPhoneValidation() *PhoneValidation {
	return &PhoneValidation{
		CountryCode: "54",
		MinLength:   10,
		MaxLength:   15,
		AllowIntl:   true,
		AreaCodes:   []string{"11", "221", "261", "264", "266", "280", "290", "291", "292", "293", "294", "295", "296", "297", "298", "299", "336", "341", "342", "343", "344", "345", "346", "347", "348", "349", "351", "352", "353", "354", "356", "357", "358", "359", "370", "371", "372", "373", "374", "375", "376", "377", "378", "379", "380", "381", "382", "383", "384", "385", "386", "387", "388", "389"},
	}
}

func (c *Config) GenerateVoucherCode() string {
	return "CH" // CheeseHouse prefix
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
