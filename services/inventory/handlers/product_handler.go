package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/Korp_Teste_PedroFrosi/inventory/models"
	"github.com/gin-gonic/gin"
)

type ProductHandler struct {
	db *sql.DB
}

func NewProductHandler(db *sql.DB) *ProductHandler {
	return &ProductHandler{db: db}
}

func (h *ProductHandler) Create(c *gin.Context) {
	var req models.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	var p models.Product
	err := h.db.QueryRow(`
		INSERT INTO products (code, description, balance)
		VALUES ($1, $2, $3)
		RETURNING id, code, description, balance, version, created_at, updated_at
	`, req.Code, req.Description, req.Balance).
		Scan(&p.ID, &p.Code, &p.Description, &p.Balance, &p.Version, &p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to create product"})
		return
	}

	c.JSON(http.StatusCreated, p)
}

func (h *ProductHandler) List(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT id, code, description, balance, version, created_at, updated_at
		FROM products ORDER BY id
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to list products"})
		return
	}
	defer rows.Close()

	products := []models.Product{}
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Code, &p.Description, &p.Balance, &p.Version, &p.CreatedAt, &p.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to scan product"})
			return
		}
		products = append(products, p)
	}

	c.JSON(http.StatusOK, products)
}

func (h *ProductHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid id"})
		return
	}

	var p models.Product
	err = h.db.QueryRow(`
		SELECT id, code, description, balance, version, created_at, updated_at
		FROM products WHERE id = $1
	`, id).Scan(&p.ID, &p.Code, &p.Description, &p.Balance, &p.Version, &p.CreatedAt, &p.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "product not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to get product"})
		return
	}

	c.JSON(http.StatusOK, p)
}

func (h *ProductHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid id"})
		return
	}

	var req models.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	var p models.Product
	err = h.db.QueryRow(`
		UPDATE products
		SET description = COALESCE(NULLIF($1, ''), description),
		    balance     = CASE WHEN $2 >= 0 THEN $2 ELSE balance END,
		    version     = version + 1
		WHERE id = $3
		RETURNING id, code, description, balance, version, created_at, updated_at
	`, req.Description, req.Balance, id).
		Scan(&p.ID, &p.Code, &p.Description, &p.Balance, &p.Version, &p.CreatedAt, &p.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "product not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to update product"})
		return
	}

	c.JSON(http.StatusOK, p)
}

func (h *ProductHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid id"})
		return
	}

	res, err := h.db.Exec(`DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to delete product"})
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "product not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

// DeductStock uses optimistic locking via version column.
func (h *ProductHandler) DeductStock(c *gin.Context) {
	var req models.DeductStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	// Read current state
	var current models.Product
	err := h.db.QueryRow(`
		SELECT id, balance, version FROM products WHERE id = $1
	`, req.ProductID).Scan(&current.ID, &current.Balance, &current.Version)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "product not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to read product"})
		return
	}

	if current.Balance < req.Quantity {
		c.JSON(http.StatusConflict, models.ErrorResponse{Error: "insufficient stock"})
		return
	}

	// Optimistic lock: only update if version hasn't changed
	var newBalance int
	err = h.db.QueryRow(`
		UPDATE products
		SET balance = balance - $1,
		    version = version + 1
		WHERE id = $2 AND version = $3
		RETURNING balance
	`, req.Quantity, req.ProductID, current.Version).Scan(&newBalance)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusConflict, models.ErrorResponse{Error: "concurrent update detected, please retry"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to deduct stock"})
		return
	}

	c.JSON(http.StatusOK, models.DeductStockResponse{
		ProductID:  req.ProductID,
		Deducted:   req.Quantity,
		NewBalance: newBalance,
	})
}