package handlers

import (
	"apis/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DocumentHandler struct {
	Service *services.DocumentService
}

func (h *DocumentHandler) Create(c *gin.Context) {
	userID := c.GetString("userID")

	var body struct {
		Title string `json:"title"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	doc, err := h.Service.Create(userID, body.Title)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, doc)
}

func (h *DocumentHandler) Update(c *gin.Context) {
	userID := c.GetString("userID")
	docID := c.Param("id")

	var body struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	doc, err := h.Service.Update(userID, docID, body.Title, body.Content)
	if err != nil {
		c.JSON(404, gin.H{"error": "document not found"})
		return
	}

	c.JSON(200, doc)
}

func (h *DocumentHandler) Get(c *gin.Context) {
	userID := c.GetString("userID")
	docID := c.Param("id")

	doc, err := h.Service.GetByID(userID, docID)
	if err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}

	c.JSON(200, doc)
}
