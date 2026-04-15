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
	"strings"
	"time"

	"github.com/Korp_Teste_PedroFrosi/billing/models"
	"github.com/Korp_Teste_PedroFrosi/billing/services"
	"github.com/gin-gonic/gin"
)

type InvoiceHandler struct {
	db              *sql.DB
	analysisSvc     *services.AnalysisService
	inventoryURL    string
	inventoryClient *http.Client
}

type stockItem struct {
	ProductID int
	Quantity  int
}

func NewInvoiceHandler(db *sql.DB) *InvoiceHandler {
	return &InvoiceHandler{
		db:              db,
		analysisSvc:     services.NewAnalysisService(),
		inventoryURL:    os.Getenv("INVENTORY_URL"),
		inventoryClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (h *InvoiceHandler) getAvailableStock(productID int) (int, error) {
	const maxRetries = 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		available, err := h.getAvailableStockOnce(productID)
		if err == nil {
			return available, nil
		}

		lastErr = err
		if !isRetryableInventoryError(err) || attempt == maxRetries {
			break
		}

		time.Sleep(time.Duration(attempt*200) * time.Millisecond)
	}

	return 0, lastErr
}

func (h *InvoiceHandler) getAvailableStockOnce(productID int) (int, error) {
	resp, err := h.inventoryClient.Get(fmt.Sprintf("%s/products/%d", h.inventoryURL, productID))
	if err != nil {
		return 0, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("inventory returned %d", resp.StatusCode)
	}

	var p struct {
		Balance int `json:"balance"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return 0, fmt.Errorf("decode error: %w", err)
	}

	var reservedQty int
	err = h.db.QueryRow(`
		SELECT COALESCE(SUM(quantity), 0)
		FROM invoice_items 
		JOIN invoices ON invoices.id = invoice_items.invoice_id 
		WHERE invoices.status = 'open' AND invoice_items.product_id = $1
	`, productID).Scan(&reservedQty)
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("fetch reserved error: %w", err)
	}

	return p.Balance - reservedQty, nil
}

// Create creates a new open invoice with items.
func (h *InvoiceHandler) Create(c *gin.Context) {
	var req models.CreateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	reqQuantities := make(map[int]int)
	for _, item := range req.Items {
		reqQuantities[item.ProductID] += item.Quantity
	}

	for pID, qty := range reqQuantities {
		available, err := h.getAvailableStock(pID)
		if err != nil {
			log.Printf("create: failed to validate stock for product %d: %v", pID, err)

			status := http.StatusBadGateway
			apiError := "failed to validate stock"
			if isInventoryUnavailableError(err) {
				status = http.StatusServiceUnavailable
				apiError = "inventory service unavailable"
			}

			c.JSON(status, models.ErrorResponse{Error: apiError})
			return
		}
		if qty > available {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error: fmt.Sprintf("insufficient stock for product ID %d: requested %d, available %d", pID, qty, available),
			})
			return
		}
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
		RETURNING id, number, status, closed_at, created_at, updated_at
	`).Scan(&invoice.ID, &invoice.Number, &invoice.Status, &invoice.ClosedAt, &invoice.CreatedAt, &invoice.UpdatedAt)
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
		SELECT id, number, status, closed_at, created_at, updated_at
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
		if err := rows.Scan(&inv.ID, &inv.Number, &inv.Status, &inv.ClosedAt, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
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
		SELECT id, number, status, closed_at, created_at, updated_at FROM invoices WHERE id = $1
	`, id).Scan(&inv.ID, &inv.Number, &inv.Status, &inv.ClosedAt, &inv.CreatedAt, &inv.UpdatedAt)

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
	var existingID, existingNumber int
	err = h.db.QueryRow(`
    	SELECT id, number FROM invoices WHERE idempotency_key = $1
	`, req.IdempotencyKey).Scan(&existingID, &existingNumber)
	if err == nil {
		c.JSON(http.StatusOK, models.PrintInvoiceResponse{
			InvoiceID:     existingID,
			InvoiceNumber: existingNumber,
			Status:        string(models.StatusClosed),
			Message:       "already processed",
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

	var items []stockItem
	for rows.Next() {
		var si stockItem
		rows.Scan(&si.ProductID, &si.Quantity)
		items = append(items, si)
	}

	// Deduct stock via batch with retry (max 3 attempts)
	const maxRetries = 3
	success := false
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		var retryable bool
		retryable, lastErr = h.deductStockBatch(items)
		if lastErr == nil {
			success = true
			break
		}
		if !retryable {
			break
		}
		log.Printf("print: batch deduct attempt %d failed: %v", attempt, lastErr)
		time.Sleep(time.Duration(attempt*200) * time.Millisecond)
	}
	if !success {
		log.Printf("print: failed to deduct stock after %d attempts: %v", maxRetries, lastErr)

		status := http.StatusBadGateway
		apiError := "failed to deduct stock"
		if isInventoryUnavailableError(lastErr) {
			status = http.StatusServiceUnavailable
			apiError = "inventory service unavailable"
		}

		c.JSON(status, models.ErrorResponse{Error: apiError})
		return
	}

	// Close invoice and save idempotency key
	_, err = h.db.Exec(`
		UPDATE invoices SET status = 'closed', closed_at = NOW(), idempotency_key = $1 WHERE id = $2
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

func (h *InvoiceHandler) deductStockBatch(items []stockItem) (bool, error) {
	var reqItems []map[string]int
	for _, item := range items {
		reqItems = append(reqItems, map[string]int{
			"product_id": item.ProductID,
			"quantity":   item.Quantity,
		})
	}

	body, _ := json.Marshal(map[string]interface{}{
		"items": reqItems,
	})

	resp, err := h.inventoryClient.Post(
		h.inventoryURL+"/stock/deduct-batch",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return true, fmt.Errorf("http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp models.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		if strings.TrimSpace(errResp.Error) == "" {
			errResp.Error = http.StatusText(resp.StatusCode)
		}

		retryable := resp.StatusCode >= http.StatusInternalServerError
		return retryable, fmt.Errorf("inventory error: %s", errResp.Error)
	}

	return false, nil
}

func isRetryableInventoryError(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())

	if strings.Contains(message, "timeout") ||
		strings.Contains(message, "deadline exceeded") ||
		strings.Contains(message, "connection refused") ||
		strings.Contains(message, "no such host") ||
		strings.Contains(message, "temporarily unavailable") {
		return true
	}

	return strings.Contains(message, "inventory returned 500") ||
		strings.Contains(message, "inventory returned 502") ||
		strings.Contains(message, "inventory returned 503") ||
		strings.Contains(message, "inventory returned 504") ||
		strings.Contains(message, "http get:") ||
		strings.Contains(message, "http error:")
}

func isInventoryUnavailableError(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())
	if strings.Contains(message, "product not found") ||
		strings.Contains(message, "insufficient stock") ||
		strings.Contains(message, "concurrent update") ||
		strings.Contains(message, "inventory returned 404") ||
		strings.Contains(message, "inventory returned 409") {
		return false
	}

	return strings.Contains(message, "inventory unavailable") || isRetryableInventoryError(err)
}

// Analyze reviews the current invoice draft with AI.
func (h *InvoiceHandler) Analyze(c *gin.Context) {
	var req models.AIAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	analysis, err := h.analysisSvc.Analyze(req.Context, req.Items)
	if err != nil {
		log.Printf("analyze: %v", err)
		c.JSON(http.StatusBadGateway, models.ErrorResponse{Error: "failed to analyze invoice: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, analysis)
}
