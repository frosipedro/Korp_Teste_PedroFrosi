package models

import "time"

type Product struct {
	ID          int       `json:"id"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	Balance     int       `json:"balance"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateProductRequest struct {
	Code        string `json:"code"        binding:"required,max=50"`
	Description string `json:"description" binding:"required,max=255"`
	Balance     int    `json:"balance"     binding:"min=0"`
}

type UpdateProductRequest struct {
	Description string `json:"description" binding:"omitempty,max=255"`
	Balance     *int    `json:"balance"     binding:"omitempty,min=0"`
}

type DeductStockRequest struct {
	ProductID int `json:"product_id" binding:"required"`
	Quantity  int `json:"quantity"   binding:"required,min=1"`
}

type DeductStockResponse struct {
	ProductID  int `json:"product_id"`
	Deducted   int `json:"deducted"`
	NewBalance int `json:"new_balance"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}