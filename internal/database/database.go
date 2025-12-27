package database

import (
	"database/sql"
	"fmt"
	"log"

	"CheeseHouse/internal/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Database struct {
	*gorm.DB
	sqlDB *sql.DB
}

func Connect(cfg *config.Config) (*Database, error) {
	// DSN para conectarse al servidor (sin seleccionar una base)
	serverDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort)

	// Nombre de la base que queremos usar/crear
	dbName := cfg.DBName
	if dbName == "" {
		dbName = "cheesehouse"
	}

	// Abrir conexión al servidor para comprobar si la base existe
	serverDB, err := gorm.Open(mysql.Open(serverDSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database server: %w", err)
	}

	// Verificar existencia de la base de datos
	exists, err := isDatabasePresent(serverDB, dbName)
	if err != nil {
		return nil, fmt.Errorf("failed to check database existence: %w", err)
	}
	if !exists {
		// Crear la base de datos si no existe
		createStmt := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;", dbName)
		if err := serverDB.Exec(createStmt).Error; err != nil {
			return nil, fmt.Errorf("failed to create database: %w", err)
		}
		log.Printf("✅ Database '%s' created (if it didn't exist)", dbName)
	}

	// Cerrar la conexión al servidor
	sqlDB, _ := serverDB.DB()
	sqlDB.Close()

	// Ahora conectamos a la base de datos específica
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, dbName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err = db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	log.Println("✅ Connected to database successfully")

	return &Database{DB: db, sqlDB: sqlDB}, nil
}

// isDatabasePresent verifica si una base de datos existe
func isDatabasePresent(db *gorm.DB, dbName string) (bool, error) {
	var exists bool
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = '%s')", dbName)
	err := db.Raw(query).Scan(&exists).Error
	return exists, err
}

func (d *Database) Health() error {
	return d.sqlDB.Ping()
}

func (d *Database) GetStats() map[string]interface{} {
	stats := d.sqlDB.Stats()
	return map[string]interface{}{
		"open_connections": stats.OpenConnections,
		"in_use":           stats.InUse,
		"idle":             stats.Idle,
		"wait_count":       stats.WaitCount,
		"wait_duration":    stats.WaitDuration.String(),
	}
}
