package handlers

import (
	"database/sql"
	"net/http"

	"aicam-backend/models"

	"github.com/gin-gonic/gin"
)

type PersonsHandler struct {
	DB *sql.DB
}

func NewPersonsHandler(db *sql.DB) *PersonsHandler {
	return &PersonsHandler{DB: db}
}

// ListPersons - GET /api/persons
func (h *PersonsHandler) ListPersons(c *gin.Context) {
	rows, err := h.DB.Query(`
		SELECT id, track_id, nome, is_critical, first_seen, last_seen 
		FROM persons 
		ORDER BY last_seen DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var persons []models.Person
	for rows.Next() {
		var p models.Person
		if err := rows.Scan(&p.ID, &p.TrackID, &p.Nome, &p.IsCritical, &p.FirstSeen, &p.LastSeen); err != nil {
			continue
		}
		persons = append(persons, p)
	}
	c.JSON(http.StatusOK, persons)
}

// ToggleCritical - PATCH /api/persons/:id
func (h *PersonsHandler) ToggleCritical(c *gin.Context) {
	id := c.Param("id")
	var input struct {
		IsCritical bool `json:"is_critical"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.DB.Exec(`
		UPDATE persons 
		SET is_critical = $1, last_seen = now() 
		WHERE id = $2`, input.IsCritical, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}
