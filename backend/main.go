package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"aicam-backend/handlers"
	"aicam-backend/middleware"
	"aicam-backend/ws"
)

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func main() {
	db := connectDB()
	defer db.Close()

	hub := ws.NewHub()

	eventsHandler := handlers.NewEventsHandler(db, hub)
	camerasHandler := handlers.NewCamerasHandler(db)
	authHandler := handlers.NewAuthHandler(db)

	router := gin.Default()
	router.Use(corsMiddleware())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// WebSocket per aggiornamenti live della dashboard
	router.GET("/ws", func(c *gin.Context) {
		hub.ServeWS(c.Writer, c.Request)
	})

	api := router.Group("/api")
	{
		auth := api.Group("/auth")
		auth.POST("/login", authHandler.Login)
		auth.POST("/register", authHandler.Register) // solo per bootstrap in demo

		// endpoint chiamato dal modulo Python: protetto da service token, non da JWT utente
		api.POST("/events", middleware.RequireServiceToken(), eventsHandler.CreateEvent)

		// endpoint di lettura per la dashboard: protetti da JWT utente
		protected := api.Group("")
		protected.Use(middleware.RequireJWT())
		{
			protected.GET("/events", eventsHandler.ListEvents)
			protected.GET("/cameras", camerasHandler.ListCameras)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Backend in ascolto sulla porta %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("errore avvio server: %v", err)
	}
}
