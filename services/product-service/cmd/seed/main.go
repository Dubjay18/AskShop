package main

import (
	"askshop/services/product-service/internal/domain"
	"askshop/services/product-service/internal/seed"
	"askshop/shared/db"
	"flag"
	"log"
	"os"

	"gorm.io/gorm"
)

func main() {
	// Parse command line flags
	var (
		clearData = flag.Bool("clear", false, "Clear all existing data before seeding")
		onlyClear = flag.Bool("clear-only", false, "Only clear data, don't seed")
	)
	flag.Parse()

	log.Println("Starting product seeding tool...")

	// Initialize database connection with auto-migration
	cfg := db.LoadConfigFromEnv()
	dbConn, err := db.Connect(cfg, db.WithAutoMigrations(func(db *gorm.DB) error {
		return db.AutoMigrate(
			&domain.Product{},
			&domain.ProductImage{},
			&domain.Category{},
		)
	}))
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize seeder
	seeder := seed.NewProductSeeder(dbConn)

	// Clear data if requested
	if *clearData || *onlyClear {
		log.Println("Clearing existing data...")
		if err := seeder.ClearData(); err != nil {
			log.Fatalf("Failed to clear data: %v", err)
		}
	}

	// Exit if only clearing
	if *onlyClear {
		log.Println("Data cleared successfully. Exiting.")
		os.Exit(0)
	}

	// Run seeding
	if err := seeder.RunAllSeeds(); err != nil {
		log.Fatalf("Seeding failed: %v", err)
	}

	log.Println("Seeding completed successfully!")
}
