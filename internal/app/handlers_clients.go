package app

import (
	"net/http"
	"traindesk/internal/DTO"
	"traindesk/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// handleCreateClient — создать нового клиента тренера.
// @Summary Create client
// @Tags clients
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body DTO.CreateClientRequest true "Client data"
// @Success 201 {object} DTO.ClientResponse
// @Failure 400 {object} DTO.ErrorResponse
// @Failure 401 {object} DTO.ErrorResponse
// @Failure 500 {object} DTO.ErrorResponse
// @Router /api/v1/clients [post]
func (a *App) handleCreateClient(c *gin.Context) {
	userIDVal, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found in context"})
		return
	}

	userIDStr, ok := userIDVal.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user_id in context"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user_id in token"})
		return
	}

	var req DTO.CreateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	if req.FirstName == "" || req.LastName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "first_name and last_name are required",
		})
		return
	}

	cl := domain.Client{
		ID:        uuid.New(),
		UserID:    userID,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}

	if err := a.db.Create(&cl).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create client"})
		return
	}

	resp := DTO.ClientResponse{
		ID:        cl.ID.String(),
		FirstName: cl.FirstName,
		LastName:  cl.LastName,
	}

	c.JSON(http.StatusCreated, resp)
}

// handleGetClients — список клиентов текущего тренера.
// @Summary List clients
// @Tags clients
// @Produce json
// @Security BearerAuth
// @Success 200 {array} DTO.ClientResponse
// @Failure 401 {object} DTO.ErrorResponse
// @Failure 500 {object} DTO.ErrorResponse
// @Router /api/v1/clients [get]
func (a *App) handleGetClients(c *gin.Context) {
	userIDVal, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found in context"})
		return
	}

	userIDStr, ok := userIDVal.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user_id in context"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user_id in token"})
		return
	}

	var clientsDB []domain.Client
	if err := a.db.Where("user_id = ?", userID).Order("last_name, first_name").Find(&clientsDB).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load clients"})
		return
	}

	resp := make([]DTO.ClientResponse, 0, len(clientsDB))
	for _, cl := range clientsDB {
		resp = append(resp, DTO.ClientResponse{
			ID:        cl.ID.String(),
			FirstName: cl.FirstName,
			LastName:  cl.LastName,
		})
	}

	c.JSON(http.StatusOK, resp)
}
