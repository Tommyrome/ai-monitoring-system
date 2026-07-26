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
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
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
	personsHandler := handlers.NewPersonsHandler(db)
	frameHandler := handlers.NewFrameHandler(hub)

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
		api.POST("/frame", middleware.RequireServiceToken(), frameHandler.PostFrame)
		// il modulo AI non ha un login utente (solo il token di servizio),
		// quindi legge la lista persone da qui invece che da /api/persons
		api.GET("/internal/persons", middleware.RequireServiceToken(), personsHandler.ListPersons)

		protected := api.Group("")
		protected.Use(middleware.RequireJWT())
		{
			protected.GET("/events", eventsHandler.ListEvents)
			protected.GET("/stats", eventsHandler.GetStats)
			protected.GET("/cameras", camerasHandler.ListCameras)
			protected.GET("/frame", frameHandler.GetFrame)
			protected.GET("/persons", personsHandler.ListPersons)
			protected.PATCH("/persons/:id", personsHandler.UpdatePerson)
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
