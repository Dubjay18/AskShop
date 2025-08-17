package db

import (
	"askshop/shared/env"
	"fmt"
	"log"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	conn    *gorm.DB
	connErr error
	once    sync.Once
)

// Config holds database configuration parameters
// DSN can be full DATABASE_URL or constructed from individual fields
// If DSN is provided it takes precedence.
type Config struct {
	DSN            string
	Host           string
	Port           int
	User           string
	Password       string
	Database       string
	SSLMode        string
	SearchPath     string
	LogLevel       logger.LogLevel
	PreferSimple   bool
	autoMigrateFns []func(db *gorm.DB) error
}

// Option functional option to modify Config
type Option func(*Config)

// WithAutoMigrations registers auto migration callbacks
func WithAutoMigrations(fns ...func(db *gorm.DB) error) Option {
	return func(c *Config) { c.autoMigrateFns = append(c.autoMigrateFns, fns...) }
}

// LoadConfigFromEnv builds Config from environment variables
// DATABASE_URL takes precedence, otherwise DB_* vars used.
func LoadConfigFromEnv() Config {
	return Config{
		DSN:          env.GetString("DATABASE_URL", ""),
		Host:         env.GetString("DB_HOST", "postgres"),
		Port:         env.GetInt("DB_PORT", 5432),
		User:         env.GetString("DB_USER", "postgres"),
		Password:     env.GetString("DB_PASSWORD", "postgres"),
		Database:     env.GetString("DB_NAME", "askshop"),
		SSLMode:      env.GetString("DB_SSLMODE", "disable"),
		SearchPath:   env.GetString("DB_SEARCH_PATH", "public"),
		LogLevel:     logger.Warn,
		PreferSimple: env.GetBool("DB_PREFER_SIMPLE", true),
	}
}

// dsn builds DSN if not explicitly provided
func (c Config) dsn() string {
	// Use DATABASE_URL if provided (takes precedence)
	if c.DSN != "" {
		log.Printf("Using DATABASE_URL for connection")
		return c.DSN
	}

	// Build DSN from individual components
	log.Printf("Building DSN from individual DB_* environment variables")
	params := fmt.Sprintf("sslmode=%s search_path=%s", c.SSLMode, c.SearchPath)
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s %s",
		c.Host, c.Port, c.User, c.Password, c.Database, params)
}

// Connect returns a singleton *gorm.DB. Subsequent calls reuse the same connection.
func Connect(cfg Config, opts ...Option) (*gorm.DB, error) {
	once.Do(func() {
		for _, o := range opts {
			o(&cfg)
		}
		gormCfg := &gorm.Config{Logger: logger.Default.LogMode(cfg.LogLevel)}
		if cfg.PreferSimple {
			gormCfg.PrepareStmt = false
		}
		conn, connErr = gorm.Open(postgres.Open(cfg.dsn()), gormCfg)
		if connErr != nil {
			return
		}
		// Run auto migrations if any
		for _, fn := range cfg.autoMigrateFns {
			if err := fn(conn); err != nil {
				log.Printf("auto migration error: %v", err)
			}
		}
	})
	return conn, connErr
}

// MustConnect wraps Connect and panics on error (for services that require DB)
func MustConnect(cfg Config, opts ...Option) *gorm.DB {
	db, err := Connect(cfg, opts...)
	if err != nil {
		panic(err)
	}
	return db
}
