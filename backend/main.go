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
	personsHandler := handlers.NewPersonsHandler(db) // ← AGGIUNTO

	router := gin.Default()
	router.Use(corsMiddleware())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	router.GET("/ws", func(c *gin.Context) {
		hub.ServeWS(c.Writer, c.Request)
	})

	api := router.Group("/api")
	{
		auth := api.Group("/auth")
		auth.POST("/login", authHandler.Login)
		auth.POST("/register", authHandler.Register)

		api.POST("/events", middleware.RequireServiceToken(), eventsHandler.CreateEvent)

		protected := api.Group("")
		protected.Use(middleware.RequireJWT())
		{
			protected.GET("/events", eventsHandler.ListEvents)
			protected.GET("/cameras", camerasHandler.ListCameras)
			protected.GET("/persons", personsHandler.ListPersons)          // ← AGGIUNTO
			protected.PATCH("/persons/:id", personsHandler.ToggleCritical) // ← AGGIUNTO
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
