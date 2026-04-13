package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Korp_Teste_PedroFrosi/billing/models"
	"github.com/Korp_Teste_PedroFrosi/billing/services"
	"github.com/gin-gonic/gin"
)

type InvoiceHandler struct {
	db          *sql.DB
	suggestionSvc *services.SuggestionService
	inventoryURL  string
}

func NewInvoiceHandler(db *sql.DB) *InvoiceHandler {
	return &InvoiceHandler{
		db:            db,
		suggestionSvc: services.NewSuggestionService(),
		inventoryURL:  os.Getenv("INVENTORY_URL"),
	}
}

// Create creates a new open invoice with items.
func (h *InvoiceHandler) Create(c *gin.Context) {
	var req models.CreateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to begin transaction"})
		return
	}
	defer tx.Rollback()

	var invoice models.Invoice
	err = tx.QueryRow(`
		INSERT INTO invoices (number)
		VALUES (nextval('invoice_number_seq'))
		RETURNING id, number, status, created_at, updated_at
	`).Scan(&invoice.ID, &invoice.Number, &invoice.Status, &invoice.CreatedAt, &invoice.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to create invoice"})
		return
	}

	for _, item := range req.Items {
		var ii models.InvoiceItem
		err = tx.QueryRow(`
			INSERT INTO invoice_items (invoice_id, product_id, product_code, description, quantity)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, invoice_id, product_id, product_code, description, quantity, created_at
		`, invoice.ID, item.ProductID, item.ProductCode, item.Description, item.Quantity).
			Scan(&ii.ID, &ii.InvoiceID, &ii.ProductID, &ii.ProductCode, &ii.Description, &ii.Quantity, &ii.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to add invoice item"})
			return
		}
		invoice.Items = append(invoice.Items, ii)
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, invoice)
}

// List returns all invoices without items.
func (h *InvoiceHandler) List(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT id, number, status, created_at, updated_at
		FROM invoices ORDER BY number DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to list invoices"})
		return
	}
	defer rows.Close()

	invoices := []models.Invoice{}
	for rows.Next() {
		var inv models.Invoice
		if err := rows.Scan(&inv.ID, &inv.Number, &inv.Status, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to scan invoice"})
			return
		}
		invoices = append(invoices, inv)
	}

	c.JSON(http.StatusOK, invoices)
}

// GetByID returns a single invoice with its items.
func (h *InvoiceHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid id"})
		return
	}

	var inv models.Invoice
	err = h.db.QueryRow(`
		SELECT id, number, status, created_at, updated_at FROM invoices WHERE id = $1
	`, id).Scan(&inv.ID, &inv.Number, &inv.Status, &inv.CreatedAt, &inv.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "invoice not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to get invoice"})
		return
	}

	rows, err := h.db.Query(`
		SELECT id, invoice_id, product_id, product_code, description, quantity, created_at
		FROM invoice_items WHERE invoice_id = $1
	`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to get items"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item models.InvoiceItem
		if err := rows.Scan(&item.ID, &item.InvoiceID, &item.ProductID, &item.ProductCode, &item.Description, &item.Quantity, &item.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to scan item"})
			return
		}
		inv.Items = append(inv.Items, item)
	}

	c.JSON(http.StatusOK, inv)
}

// Print closes an invoice and deducts stock with idempotency + retry.
func (h *InvoiceHandler) Print(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid id"})
		return
	}

	var req models.PrintInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	// Idempotency check
	var existingID int
	err = h.db.QueryRow(`
		SELECT id FROM invoices WHERE idempotency_key = $1
	`, req.IdempotencyKey).Scan(&existingID)
	if err == nil {
		c.JSON(http.StatusOK, models.PrintInvoiceResponse{
			InvoiceID: existingID,
			Status:    string(models.StatusClosed),
			Message:   "already processed",
		})
		return
	}

	// Load invoice
	var inv models.Invoice
	err = h.db.QueryRow(`
		SELECT id, number, status FROM invoices WHERE id = $1
	`, id).Scan(&inv.ID, &inv.Number, &inv.Status)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "invoice not found"})
		return
	}
	if inv.Status == models.StatusClosed {
		c.JSON(http.StatusConflict, models.ErrorResponse{Error: "invoice already closed"})
		return
	}

	// Load items
	rows, err := h.db.Query(`
		SELECT product_id, quantity FROM invoice_items WHERE invoice_id = $1
	`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to load items"})
		return
	}
	defer rows.Close()

	type stockItem struct {
		ProductID int
		Quantity  int
	}
	var items []stockItem
	for rows.Next() {
		var si stockItem
		rows.Scan(&si.ProductID, &si.Quantity)
		items = append(items, si)
	}

	// Deduct stock with retry (max 3 attempts)
	const maxRetries = 3
	for _, item := range items {
		success := false
		var lastErr error
		for attempt := 1; attempt <= maxRetries; attempt++ {
			lastErr = h.deductStock(item.ProductID, item.Quantity)
			if lastErr == nil {
				success = true
				break
			}
			log.Printf("print: deduct attempt %d failed for product %d: %v", attempt, item.ProductID, lastErr)
			time.Sleep(time.Duration(attempt*200) * time.Millisecond)
		}
		if !success {
			c.JSON(http.StatusBadGateway, models.ErrorResponse{
				Error: fmt.Sprintf("failed to deduct stock for product %d after %d attempts: %v", item.ProductID, maxRetries, lastErr),
			})
			return
		}
	}

	// Close invoice and save idempotency key
	_, err = h.db.Exec(`
		UPDATE invoices SET status = 'closed', idempotency_key = $1 WHERE id = $2
	`, req.IdempotencyKey, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to close invoice"})
		return
	}

	c.JSON(http.StatusOK, models.PrintInvoiceResponse{
		InvoiceID:     inv.ID,
		InvoiceNumber: inv.Number,
		Status:        string(models.StatusClosed),
		Message:       "invoice printed successfully",
	})
}

func (h *InvoiceHandler) deductStock(productID, quantity int) error {
	body, _ := json.Marshal(map[string]int{
		"product_id": productID,
		"quantity":   quantity,
	})

	resp, err := http.Post(
		h.inventoryURL+"/stock/deduct",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp models.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("inventory error: %s", errResp.Error)
	}

	return nil
}

// Suggest calls the AI suggestion service.
func (h *InvoiceHandler) Suggest(c *gin.Context) {
	var req models.AISuggestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	suggestions, err := h.suggestionSvc.Suggest(req.Description)
	if err != nil {
		log.Printf("suggest: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to get suggestions"})
		return
	}

	c.JSON(http.StatusOK, models.AISuggestionResponse{Suggestions: suggestions})
}