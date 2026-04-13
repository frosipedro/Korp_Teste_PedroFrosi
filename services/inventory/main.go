package main

import (
	"log"
	"os"

	"github.com/Korp_Teste_PedroFrosi/inventory/db"
	"github.com/Korp_Teste_PedroFrosi/inventory/handlers"
	"github.com/gin-gonic/gin"
)

func main() {
	database := db.Connect()
	defer database.Close()

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	h := handlers.NewProductHandler(database)

	products := r.Group("/products")
	{
		products.POST("", h.Create)
		products.GET("", h.List)
		products.GET("/:id", h.GetByID)
		products.PUT("/:id", h.Update)
		products.DELETE("/:id", h.Delete)
	}

	r.POST("/stock/deduct", h.DeductStock)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("inventory service running on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
