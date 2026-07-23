package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"aicam-backend/middleware"
)

type AuthHandler struct {
	DB *sql.DB
}

func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{DB: db}
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login - POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username e password richiesti"})
		return
	}

	var id, hash, ruolo string
	err := h.DB.QueryRow(
		`SELECT id, password_hash, ruolo FROM users WHERE username = $1`, req.Username,
	).Scan(&id, &hash, &ruolo)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "credenziali non valide"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "credenziali non valide"})
		return
	}

	token, err := middleware.GenerateToken(id, req.Username, ruolo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "errore generazione token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token, "username": req.Username, "ruolo": ruolo})
}

// Register - POST /api/auth/register (utile solo per bootstrap/demo)
func (h *AuthHandler) Register(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username e password richiesti"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "errore hashing password"})
		return
	}

	_, err = h.DB.Exec(
		`INSERT INTO users (username, password_hash, ruolo) VALUES ($1, $2, 'operator')`,
		req.Username, string(hash),
	)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "utente gia' esistente o errore: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "utente creato"})
}
