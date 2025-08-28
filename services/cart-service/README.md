# Cart Service

The Cart Service is a microservice responsible for managing shopping carts, cart items, and saved items (wishlist) in the AskShop e-commerce platform.

## Features

### 🛒 Cart Management
- **Create/Get Cart**: Automatic cart creation for users and anonymous sessions
- **Cart Persistence**: Carts persist across sessions with configurable expiry
- **Cart Merging**: Seamless merging of anonymous carts when users log in
- **Cart Status**: Support for active, abandoned, converted, and expired states

### 📦 Item Management
- **Add Items**: Add products to cart with quantity and variations (size, color, etc.)
- **Update Quantity**: Modify item quantities with validation
- **Remove Items**: Remove individual items or clear entire cart
- **Duplicate Detection**: Automatically merge items with same product and variations

### 💾 Wishlist/Save for Later
- **Save Items**: Move cart items to saved items for later purchase
- **Manage Wishlist**: Full CRUD operations on saved items
- **Priority Support**: Organize saved items by priority

### 💰 Pricing & Calculations
- **Real-time Totals**: Automatic calculation of subtotals, discounts, taxes, shipping
- **Discount Support**: Item-level and cart-level discount handling
- **Multi-currency**: Support for different currencies
- **Tax Calculation**: Configurable tax rates

### 🔄 Advanced Features
- **Cart Validation**: Comprehensive validation before checkout
- **Price Change Detection**: Alert users to price changes since items were added
- **Abandoned Cart Recovery**: Track and recover abandoned carts
- **Analytics**: Cart statistics and conversion tracking

## Architecture

The service follows Clean Architecture principles with the following structure:

```
services/cart-service/
├── cmd/                    # Application entry points
│   └── main.go            # Main application setup
├── internal/              # Private application code
│   ├── domain/           # Business domain models and interfaces
│   │   ├── cart.go       # Cart domain models
│   │   ├── repository.go # Repository interfaces
│   │   ├── service.go    # Service interfaces
│   │   └── cart_logic.go # Business logic and validation
│   ├── service/          # Business logic implementation
│   │   └── cart_service.go # Cart service implementation
│   └── infrastructure/   # External dependencies implementations
│       ├── events/       # Event handling (RabbitMQ)
│       ├── grpc/         # gRPC server handlers
│       ├── http/         # HTTP handlers
│       └── repository/   # Data persistence
├── pkg/                  # Public packages
│   └── types/           # Shared types and models
└── README.md            # This file
```

## Database Schema

### Core Tables

#### `carts`
- `id` (UUID) - Primary key
- `user_id` (VARCHAR) - User identifier
- `session_id` (VARCHAR) - Session for anonymous carts
- `status` (VARCHAR) - Cart status (active, abandoned, converted, expired)
- `currency` (CHAR(3)) - Currency code (USD, EUR, etc.)
- `notes` (TEXT) - Cart notes
- `metadata` (JSONB) - Additional cart data
- `tags` (JSONB) - Cart tags array
- `ip_address` (VARCHAR) - User IP for analytics
- `expires_at` (TIMESTAMP) - Cart expiration time
- `created_at`, `updated_at`, `deleted_at` - Timestamps

#### `cart_items`
- `id` (UUID) - Primary key
- `cart_id` (UUID) - Foreign key to carts
- `product_id` (UUID) - Product reference
- `product_sku` (VARCHAR) - Product SKU snapshot
- `product_name` (VARCHAR) - Product name snapshot
- `unit_price` (DECIMAL) - Price snapshot
- `currency` (CHAR(3)) - Currency code
- `discount_rate` (DECIMAL) - Discount percentage
- `quantity` (INT) - Item quantity
- `variations` (JSONB) - Product variations (size, color, etc.)
- `notes` (TEXT) - Item notes
- `metadata` (JSONB) - Additional item data
- `created_at`, `updated_at` - Timestamps

#### `saved_items`
- `id` (UUID) - Primary key
- `user_id` (VARCHAR) - User identifier
- `product_id` (UUID) - Product reference
- `product_name` (VARCHAR) - Product name snapshot
- `product_sku` (VARCHAR) - Product SKU snapshot
- `unit_price` (DECIMAL) - Price snapshot
- `currency` (CHAR(3)) - Currency code
- `variations` (JSONB) - Product variations
- `tags` (JSONB) - Item tags (wishlist, compare, etc.)
- `notes` (TEXT) - Item notes
- `metadata` (JSONB) - Additional data
- `priority` (INT) - Item priority in wishlist
- `created_at`, `updated_at`, `deleted_at` - Timestamps

## Key Benefits

1. **Dependency Inversion**: Services depend on interfaces, not implementations
2. **Separation of Concerns**: Each layer has a specific responsibility
3. **Testability**: Easy to mock dependencies for testing
4. **Maintainability**: Clear boundaries between components
5. **Flexibility**: Easy to swap implementations without affecting business logic
6. **Scalability**: Supports horizontal scaling and high availability
7. **Business Logic Isolation**: Pure business rules separated from infrastructure concerns
