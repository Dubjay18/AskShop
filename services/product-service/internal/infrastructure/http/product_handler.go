package http

import (
	"askshop/services/product-service/internal/domain"
	"askshop/shared/contracts"
	"askshop/shared/response"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProductResponse is the wire format returned to clients.
type ProductResponse struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Slug          string   `json:"slug"`
	SKU           string   `json:"sku"`
	Description   string   `json:"description"`
	Currency      string   `json:"currency"`
	Status        string   `json:"status"`
	Tags          []string `json:"tags"`
	PriceCents    int64    `json:"priceCents"`
	StockQuantity int      `json:"stockQuantity"`
}

func toProductResponse(p *domain.Product) ProductResponse {
	return ProductResponse{
		ID:            p.ID.String(),
		Name:          p.Name,
		Slug:          p.Slug,
		SKU:           p.SKU,
		Description:   p.Description,
		Currency:      p.Currency,
		Status:        p.Status,
		Tags:          append([]string(nil), p.Tags...),
		PriceCents:    p.PriceCents,
		StockQuantity: p.StockQuantity,
	}
}

// CreateProductRequest is the payload for creating a product.
type CreateProductRequest struct {
	Name          string   `json:"name" binding:"required"`
	SKU           string   `json:"sku" binding:"required"`
	Description   string   `json:"description"`
	Currency      string   `json:"currency"`
	Status        string   `json:"status"`
	Tags          []string `json:"tags"`
	PriceCents    int64    `json:"priceCents" binding:"min=0"`
	StockQuantity int      `json:"stockQuantity" binding:"min=0"`
}

// UpdateProductRequest is the payload for updating a product. Zero-value fields are
// left as sent (this is a full-replace update, not a partial patch).
type UpdateProductRequest struct {
	Name          string   `json:"name" binding:"required"`
	SKU           string   `json:"sku" binding:"required"`
	Description   string   `json:"description"`
	Currency      string   `json:"currency"`
	Status        string   `json:"status"`
	Tags          []string `json:"tags"`
	PriceCents    int64    `json:"priceCents" binding:"min=0"`
	StockQuantity int      `json:"stockQuantity" binding:"min=0"`
}

type ProductHandler struct {
	service domain.ProductService
}

func NewProductHandler(service domain.ProductService) *ProductHandler {
	return &ProductHandler{service: service}
}

// RegisterRoutes wires product endpoints under /api/v1/products.
func (h *ProductHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	products := api.Group("/products")
	{
		products.GET("", h.ListProducts)
		products.POST("", h.CreateProduct)
		products.GET("/:id", h.GetProduct)
		products.PUT("/:id", h.UpdateProduct)
		products.DELETE("/:id", h.DeleteProduct)
	}
}

func mapDomainError(err error) (int, string, string, interface{}) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return http.StatusNotFound, contracts.CodeInternalServerError, "Product not found", nil
	default:
		return http.StatusInternalServerError, contracts.CodeInternalServerError, "Internal server error", err.Error()
	}
}

// ListProducts handles GET /api/v1/products, optionally filtered by category and paginated.
func (h *ProductHandler) ListProducts(c *gin.Context) {
	category := c.Query("category")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if pageSize <= 0 {
		pageSize = 20
	}

	var (
		products []*domain.Product
		total    int64
		err      error
	)
	if category != "" {
		products, total, err = h.service.GetProductsByCategoryPage(c.Request.Context(), category, page, pageSize)
	} else {
		products, total, err = h.service.GetProductsPage(c.Request.Context(), page, pageSize)
	}
	if err != nil {
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}

	out := make([]ProductResponse, 0, len(products))
	for _, p := range products {
		out = append(out, toProductResponse(p))
	}
	response.Success(c, http.StatusOK, out, gin.H{"total": total, "page": page, "pageSize": pageSize}, "")
}

// GetProduct handles GET /api/v1/products/:id.
func (h *ProductHandler) GetProduct(c *gin.Context) {
	id := c.Param("id")
	product, err := h.service.GetProductByID(c.Request.Context(), id)
	if err != nil {
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}
	response.Success(c, http.StatusOK, toProductResponse(product), nil, "")
}

// CreateProduct handles POST /api/v1/products.
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, contracts.CodeInvalidRequestBody, "Invalid request body", err.Error())
		return
	}

	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}
	status := req.Status
	if status == "" {
		status = "draft"
	}

	product := &domain.Product{
		Name:          req.Name,
		SKU:           req.SKU,
		Description:   req.Description,
		Currency:      currency,
		Status:        status,
		Tags:          req.Tags,
		PriceCents:    req.PriceCents,
		StockQuantity: req.StockQuantity,
	}

	created, err := h.service.CreateProduct(c.Request.Context(), product)
	if err != nil {
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}
	response.Success(c, http.StatusCreated, toProductResponse(created), nil, "Product created successfully")
}

// UpdateProduct handles PUT /api/v1/products/:id.
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(c, http.StatusBadRequest, contracts.CodeInvalidRequestBody, "Invalid product id", nil)
		return
	}

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, contracts.CodeInvalidRequestBody, "Invalid request body", err.Error())
		return
	}

	product := &domain.Product{
		ID:            id,
		Name:          req.Name,
		SKU:           req.SKU,
		Description:   req.Description,
		Currency:      req.Currency,
		Status:        req.Status,
		Tags:          req.Tags,
		PriceCents:    req.PriceCents,
		StockQuantity: req.StockQuantity,
	}

	updated, err := h.service.UpdateProduct(c.Request.Context(), product)
	if err != nil {
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}
	response.Success(c, http.StatusOK, toProductResponse(updated), nil, "Product updated successfully")
}

// DeleteProduct handles DELETE /api/v1/products/:id.
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(c, http.StatusBadRequest, contracts.CodeInvalidRequestBody, "Invalid product id", nil)
		return
	}

	if err := h.service.DeleteProduct(c.Request.Context(), id); err != nil {
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}
	response.Success(c, http.StatusOK, nil, nil, "Product deleted successfully")
}
