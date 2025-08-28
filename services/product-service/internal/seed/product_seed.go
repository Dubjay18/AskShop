package seed

import (
	"askshop/services/product-service/internal/domain"
	"log"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ProductSeeder handles seeding product data
type ProductSeeder struct {
	db *gorm.DB
}

// NewProductSeeder creates a new product seeder
func NewProductSeeder(db *gorm.DB) *ProductSeeder {
	return &ProductSeeder{db: db}
}

// SeedCategories creates sample categories
func (s *ProductSeeder) SeedCategories() error {
	categories := []domain.Category{
		{
			ID:   uuid.New(),
			Name: "Electronics",
			Slug: "electronics",
		},
		{
			ID:   uuid.New(),
			Name: "Clothing",
			Slug: "clothing",
		},
		{
			ID:   uuid.New(),
			Name: "Books",
			Slug: "books",
		},
		{
			ID:   uuid.New(),
			Name: "Home & Garden",
			Slug: "home-garden",
		},
		{
			ID:   uuid.New(),
			Name: "Sports & Outdoors",
			Slug: "sports-outdoors",
		},
		{
			ID:   uuid.New(),
			Name: "Health & Beauty",
			Slug: "health-beauty",
		},
	}

	// Create subcategories
	electronicsID := categories[0].ID
	clothingID := categories[1].ID

	subcategories := []domain.Category{
		{
			ID:       uuid.New(),
			Name:     "Smartphones",
			Slug:     "smartphones",
			ParentID: &electronicsID,
		},
		{
			ID:       uuid.New(),
			Name:     "Laptops",
			Slug:     "laptops",
			ParentID: &electronicsID,
		},
		{
			ID:       uuid.New(),
			Name:     "Headphones",
			Slug:     "headphones",
			ParentID: &electronicsID,
		},
		{
			ID:       uuid.New(),
			Name:     "Men's Clothing",
			Slug:     "mens-clothing",
			ParentID: &clothingID,
		},
		{
			ID:       uuid.New(),
			Name:     "Women's Clothing",
			Slug:     "womens-clothing",
			ParentID: &clothingID,
		},
	}

	categories = append(categories, subcategories...)

	for _, category := range categories {
		result := s.db.FirstOrCreate(&category, domain.Category{Slug: category.Slug})
		if result.Error != nil {
			log.Printf("Error creating category %s: %v", category.Name, result.Error)
			return result.Error
		}
	}

	log.Printf("Successfully seeded %d categories", len(categories))
	return nil
}

// SeedProducts creates sample products
func (s *ProductSeeder) SeedProducts() error {
	// Get category IDs for associations
	var electronics, smartphones, laptops, headphones, mensClothing, womensClothing domain.Category
	s.db.Where("slug = ?", "electronics").First(&electronics)
	s.db.Where("slug = ?", "smartphones").First(&smartphones)
	s.db.Where("slug = ?", "laptops").First(&laptops)
	s.db.Where("slug = ?", "headphones").First(&headphones)
	s.db.Where("slug = ?", "mens-clothing").First(&mensClothing)
	s.db.Where("slug = ?", "womens-clothing").First(&womensClothing)

	products := []struct {
		Product    domain.Product
		Categories []domain.Category
		Images     []domain.ProductImage
	}{
		{
			Product: domain.Product{
				ID:          uuid.New(),
				Name:        "iPhone 15 Pro",
				Slug:        "iphone-15-pro",
				SKU:         "IPH15PRO-128-TIT",
				Description: "The latest iPhone with titanium design and advanced camera system. Features the powerful A17 Pro chip and enhanced battery life.",
				Currency:    "USD",
				Status:      "active",
				Tags:        datatypes.JSONSlice[string]{"new", "featured", "smartphone", "apple"},
			},
			Categories: []domain.Category{electronics, smartphones},
			Images: []domain.ProductImage{
				{
					ID:       uuid.New(),
					URL:      "https://example.com/images/iphone-15-pro-front.jpg",
					Alt:      "iPhone 15 Pro front view",
					Position: 1,
					Metadata: datatypes.JSONMap{
						"width":  "800",
						"height": "800",
						"format": "jpg",
					},
				},
				{
					ID:       uuid.New(),
					URL:      "https://example.com/images/iphone-15-pro-back.jpg",
					Alt:      "iPhone 15 Pro back view",
					Position: 2,
					Metadata: datatypes.JSONMap{
						"width":  "800",
						"height": "800",
						"format": "jpg",
					},
				},
			},
		},
		{
			Product: domain.Product{
				ID:          uuid.New(),
				Name:        "MacBook Pro 16-inch",
				Slug:        "macbook-pro-16-inch",
				SKU:         "MBP16-M3-512-SG",
				Description: "Professional laptop with M3 chip, 16-inch Liquid Retina display, and up to 22 hours of battery life. Perfect for creative professionals.",
				Currency:    "USD",
				Status:      "active",
				Tags:        datatypes.JSONSlice[string]{"laptop", "professional", "apple", "m3"},
			},
			Categories: []domain.Category{electronics, laptops},
			Images: []domain.ProductImage{
				{
					ID:       uuid.New(),
					URL:      "https://example.com/images/macbook-pro-16-open.jpg",
					Alt:      "MacBook Pro 16-inch open view",
					Position: 1,
					Metadata: datatypes.JSONMap{
						"width":  "1200",
						"height": "800",
						"format": "jpg",
					},
				},
				{
					ID:       uuid.New(),
					URL:      "https://example.com/images/macbook-pro-16-closed.jpg",
					Alt:      "MacBook Pro 16-inch closed view",
					Position: 2,
					Metadata: datatypes.JSONMap{
						"width":  "1200",
						"height": "800",
						"format": "jpg",
					},
				},
			},
		},
		{
			Product: domain.Product{
				ID:          uuid.New(),
				Name:        "Sony WH-1000XM5 Headphones",
				Slug:        "sony-wh-1000xm5-headphones",
				SKU:         "SONY-WH1000XM5-BLK",
				Description: "Industry-leading noise canceling with premium sound quality. 30-hour battery life and quick charge feature.",
				Currency:    "USD",
				Status:      "active",
				Tags:        datatypes.JSONSlice[string]{"headphones", "wireless", "noise-canceling", "sony"},
			},
			Categories: []domain.Category{electronics, headphones},
			Images: []domain.ProductImage{
				{
					ID:       uuid.New(),
					URL:      "https://example.com/images/sony-wh1000xm5-side.jpg",
					Alt:      "Sony WH-1000XM5 side view",
					Position: 1,
					Metadata: datatypes.JSONMap{
						"width":  "800",
						"height": "800",
						"format": "jpg",
					},
				},
			},
		},
		{
			Product: domain.Product{
				ID:          uuid.New(),
				Name:        "Classic Denim Jacket",
				Slug:        "classic-denim-jacket",
				SKU:         "CDJ-MEN-L-BLUE",
				Description: "Timeless denim jacket made from premium cotton. Perfect for casual wear and layering. Available in multiple sizes.",
				Currency:    "USD",
				Status:      "active",
				Tags:        datatypes.JSONSlice[string]{"jacket", "denim", "casual", "classic"},
			},
			Categories: []domain.Category{mensClothing},
			Images: []domain.ProductImage{
				{
					ID:       uuid.New(),
					URL:      "https://example.com/images/denim-jacket-front.jpg",
					Alt:      "Classic denim jacket front view",
					Position: 1,
					Metadata: datatypes.JSONMap{
						"width":  "600",
						"height": "800",
						"format": "jpg",
					},
				},
				{
					ID:       uuid.New(),
					URL:      "https://example.com/images/denim-jacket-back.jpg",
					Alt:      "Classic denim jacket back view",
					Position: 2,
					Metadata: datatypes.JSONMap{
						"width":  "600",
						"height": "800",
						"format": "jpg",
					},
				},
			},
		},
		{
			Product: domain.Product{
				ID:          uuid.New(),
				Name:        "Elegant Summer Dress",
				Slug:        "elegant-summer-dress",
				SKU:         "ESD-WOM-M-FLO",
				Description: "Beautiful floral summer dress made from breathable fabric. Perfect for warm weather and special occasions.",
				Currency:    "USD",
				Status:      "active",
				Tags:        datatypes.JSONSlice[string]{"dress", "summer", "floral", "elegant"},
			},
			Categories: []domain.Category{womensClothing},
			Images: []domain.ProductImage{
				{
					ID:       uuid.New(),
					URL:      "https://example.com/images/summer-dress-front.jpg",
					Alt:      "Elegant summer dress front view",
					Position: 1,
					Metadata: datatypes.JSONMap{
						"width":  "600",
						"height": "900",
						"format": "jpg",
					},
				},
			},
		},
		{
			Product: domain.Product{
				ID:          uuid.New(),
				Name:        "Wireless Gaming Mouse",
				Slug:        "wireless-gaming-mouse",
				SKU:         "WGM-RGB-BLK-PRO",
				Description: "High-precision wireless gaming mouse with customizable RGB lighting and programmable buttons. 50-hour battery life.",
				Currency:    "USD",
				Status:      "active",
				Tags:        datatypes.JSONSlice[string]{"gaming", "mouse", "wireless", "rgb"},
			},
			Categories: []domain.Category{electronics},
			Images: []domain.ProductImage{
				{
					ID:       uuid.New(),
					URL:      "https://example.com/images/gaming-mouse-top.jpg",
					Alt:      "Wireless gaming mouse top view",
					Position: 1,
					Metadata: datatypes.JSONMap{
						"width":  "800",
						"height": "600",
						"format": "jpg",
					},
				},
			},
		},
		// Draft product for testing
		{
			Product: domain.Product{
				ID:          uuid.New(),
				Name:        "Upcoming Product",
				Slug:        "upcoming-product",
				SKU:         "UP-DRAFT-001",
				Description: "This is a draft product that will be released soon. Stay tuned for updates!",
				Currency:    "USD",
				Status:      "draft",
				Tags:        datatypes.JSONSlice[string]{"coming-soon", "draft"},
			},
			Categories: []domain.Category{electronics},
			Images:     []domain.ProductImage{},
		},
	}

	for _, productData := range products {
		// Check if product already exists
		var existingProduct domain.Product
		result := s.db.Where("sku = ?", productData.Product.SKU).First(&existingProduct)
		if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
			return result.Error
		}

		// Skip if product already exists
		if result.Error == nil {
			log.Printf("Product with SKU %s already exists, skipping", productData.Product.SKU)
			continue
		}

		// Create product
		if err := s.db.Create(&productData.Product).Error; err != nil {
			log.Printf("Error creating product %s: %v", productData.Product.Name, err)
			return err
		}

		// Associate categories
		if len(productData.Categories) > 0 {
			if err := s.db.Model(&productData.Product).Association("Categories").Append(productData.Categories); err != nil {
				log.Printf("Error associating categories for product %s: %v", productData.Product.Name, err)
				return err
			}
		}

		// Create product images
		for i := range productData.Images {
			productData.Images[i].ProductID = productData.Product.ID
			if err := s.db.Create(&productData.Images[i]).Error; err != nil {
				log.Printf("Error creating image for product %s: %v", productData.Product.Name, err)
				return err
			}
		}

		log.Printf("Successfully created product: %s", productData.Product.Name)
	}

	return nil
}

// RunAllSeeds runs all seeding operations
func (s *ProductSeeder) RunAllSeeds() error {
	log.Println("Starting product seeding...")

	// Seed categories first
	if err := s.SeedCategories(); err != nil {
		return err
	}

	// Then seed products
	if err := s.SeedProducts(); err != nil {
		return err
	}

	log.Println("Product seeding completed successfully!")
	return nil
}

// ClearData removes all seeded data (useful for testing)
func (s *ProductSeeder) ClearData() error {
	// Delete in reverse order of dependencies
	if err := s.db.Exec("DELETE FROM product_images").Error; err != nil {
		return err
	}

	if err := s.db.Exec("DELETE FROM product_categories").Error; err != nil {
		return err
	}

	if err := s.db.Exec("DELETE FROM products").Error; err != nil {
		return err
	}

	if err := s.db.Exec("DELETE FROM categories").Error; err != nil {
		return err
	}

	log.Println("All seeded data cleared successfully!")
	return nil
}
