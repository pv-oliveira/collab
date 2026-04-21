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
	var body struct {
		Title string `json:"title"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	doc, err := h.Service.Create(body.Title)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, doc)
}
