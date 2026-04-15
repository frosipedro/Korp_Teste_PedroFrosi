package main

import (
	"log"
	"os"

	"github.com/Korp_Teste_PedroFrosi/billing/db"
	"github.com/Korp_Teste_PedroFrosi/billing/handlers"
	"github.com/gin-gonic/gin"
)

func main() {
	database := db.Connect()
	defer database.Close()

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	h := handlers.NewInvoiceHandler(database)

	invoices := r.Group("/invoices")
	{
		invoices.POST("", h.Create)
		invoices.GET("", h.List)
		invoices.GET("/:id", h.GetByID)
		invoices.POST("/:id/print", h.Print)
	}

	r.POST("/ai/analyze", h.Analyze)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("billing service running on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}