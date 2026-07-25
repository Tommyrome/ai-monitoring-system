package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

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

	persons := make([]models.Person, 0)
	for rows.Next() {
		var p models.Person
		if err := rows.Scan(&p.ID, &p.TrackID, &p.Nome, &p.IsCritical, &p.FirstSeen, &p.LastSeen); err != nil {
			continue
		}
		persons = append(persons, p)
	}
	c.JSON(http.StatusOK, persons)
}

type updatePersonInput struct {
	Nome       *string `json:"nome"`
	IsCritical *bool   `json:"is_critical"`
}

// UpdatePerson - PATCH /api/persons/:id
// Supporta l'aggiornamento parziale: rinomina la persona, imposta lo
// stato "critica", o entrambi in un'unica richiesta.
func (h *PersonsHandler) UpdatePerson(c *gin.Context) {
	id := c.Param("id")
	var input updatePersonInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Nome == nil && input.IsCritical == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "specificare almeno nome o is_critical"})
		return
	}

	if input.Nome != nil && strings.TrimSpace(*input.Nome) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "il nome non puo' essere vuoto"})
		return
	}

	setClauses := []string{}
	args := []interface{}{}
	argN := 1

	if input.Nome != nil {
		setClauses = append(setClauses, "nome = $"+strconv.Itoa(argN))
		args = append(args, strings.TrimSpace(*input.Nome))
		argN++
	}
	if input.IsCritical != nil {
		setClauses = append(setClauses, "is_critical = $"+strconv.Itoa(argN))
		args = append(args, *input.IsCritical)
		argN++
	}

	query := "UPDATE persons SET " + strings.Join(setClauses, ", ") + " WHERE id = $" + strconv.Itoa(argN) +
		" RETURNING id, track_id, nome, is_critical, first_seen, last_seen"
	args = append(args, id)

	var p models.Person
	err := h.DB.QueryRow(query, args...).Scan(&p.ID, &p.TrackID, &p.Nome, &p.IsCritical, &p.FirstSeen, &p.LastSeen)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "persona non trovata"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, p)
}
