package models

import "time"

type InvoiceStatus string

const (
	StatusOpen   InvoiceStatus = "open"
	StatusClosed InvoiceStatus = "closed"
)

type Invoice struct {
	ID             int           `json:"id"`
	Number         int           `json:"number"`
	Status         InvoiceStatus `json:"status"`
	ClosedAt       *time.Time    `json:"closed_at,omitempty"`
	IdempotencyKey string        `json:"idempotency_key,omitempty"`
	Items          []InvoiceItem `json:"items,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

type InvoiceItem struct {
	ID          int       `json:"id"`
	InvoiceID   int       `json:"invoice_id"`
	ProductID   int       `json:"product_id"`
	ProductCode string    `json:"product_code"`
	Description string    `json:"description"`
	Quantity    int       `json:"quantity"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateInvoiceRequest struct {
	Items []CreateInvoiceItemRequest `json:"items" binding:"required,min=1,dive"`
}

type CreateInvoiceItemRequest struct {
	ProductID   int    `json:"product_id"   binding:"required"`
	ProductCode string `json:"product_code" binding:"required,max=50"`
	Description string `json:"description"  binding:"required,max=255"`
	Quantity    int    `json:"quantity"     binding:"required,min=1"`
}

type PrintInvoiceRequest struct {
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
}

type PrintInvoiceResponse struct {
	InvoiceID     int    `json:"invoice_id"`
	InvoiceNumber int    `json:"invoice_number"`
	Status        string `json:"status"`
	Message       string `json:"message"`
}

type AIAnalysisItem struct {
	ProductID   int    `json:"product_id" binding:"required"`
	ProductCode string `json:"product_code" binding:"required,max=50"`
	Description string `json:"description" binding:"required,max=255"`
	Quantity    int    `json:"quantity" binding:"required,min=1"`
}

type AIAnalysisRequest struct {
	Context string           `json:"context,omitempty"`
	Items   []AIAnalysisItem `json:"items" binding:"required,min=1,dive"`
}

type AIAnalysisResponse struct {
	Summary         string   `json:"summary"`
	Category        string   `json:"category"`
	RiskLevel       string   `json:"risk_level"`
	Alerts          []string `json:"alerts"`
	Recommendations []string `json:"recommendations"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}