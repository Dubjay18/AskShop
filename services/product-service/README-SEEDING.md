# Product Service Seeding

This directory contains seeding utilities for the product service database.

## Overview

The seeding system creates sample data for:
- **Categories**: Electronics, Clothing, Books, Home & Garden, Sports & Outdoors, Health & Beauty
- **Subcategories**: Smartphones, Laptops, Headphones, Men's/Women's Clothing
- **Products**: Sample products with realistic data including images and metadata
- **Product Images**: Sample product images with metadata

## Usage

### Running the Seeder

```bash
# From project root
./scripts/seed-products.sh

# Or build and run directly
go build -o build/product-seeder ./services/product-service/cmd/seed/
./build/product-seeder
```

### Command Line Options

```bash
# Clear existing data before seeding
./build/product-seeder -clear

# Only clear data (don't seed)
./build/product-seeder -clear-only
```

## Sample Data

The seeder creates:

### Categories
- Electronics (with subcategories: Smartphones, Laptops, Headphones)
- Clothing (with subcategories: Men's Clothing, Women's Clothing)
- Books
- Home & Garden
- Sports & Outdoors
- Health & Beauty

### Products
1. **iPhone 15 Pro** - Premium smartphone with advanced features
2. **MacBook Pro 16-inch** - Professional laptop with M3 chip
3. **Sony WH-1000XM5 Headphones** - Noise-canceling headphones
4. **Classic Denim Jacket** - Timeless men's fashion item
5. **Elegant Summer Dress** - Women's floral dress
6. **Wireless Gaming Mouse** - High-precision gaming peripheral
7. **Upcoming Product** - Draft status product for testing

### Product Features
- **UUID Primary Keys**: All products use UUID for better distribution
- **SEO-Friendly Slugs**: Auto-generated from product names
- **SKU System**: Structured SKU codes for inventory management
- **Tags**: JSON array of searchable tags
- **Status System**: Active, draft, and archived product states
- **Image Galleries**: Multiple images per product with metadata
- **Category Associations**: Many-to-many relationships

## Database Schema

The seeder automatically runs migrations for:
- `products` table
- `product_images` table
- `categories` table
- `product_categories` junction table

## Environment Setup

Make sure your database connection is configured in your `.env` file:

```env
DATABASE_URL=postgresql://username:password@host:port/database
```

## Development

To modify the seed data:
1. Edit `/services/product-service/internal/seed/product_seed.go`
2. Add new products to the `products` slice in `SeedProducts()`
3. Rebuild and run the seeder

## Production Notes

⚠️ **Warning**: The `ClearData()` function will delete all products, categories, and images. Use with caution in production environments.

For production seeding:
- Use the seeder only once during initial deployment
- Consider using database backups before running
- Test thoroughly in staging environments first
