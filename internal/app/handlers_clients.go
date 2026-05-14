package app

import (
	"net/http"
	"strings"
	"traindesk/internal/DTO"
	"traindesk/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type clientSessionStats struct {
	ClientID      uuid.UUID
	TotalSessions int
	LeftSessions  int
}

// handleCreateClient creates a new trainer client.
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
	userID := uuid.MustParse(c.MustGet("user_id").(string))

	var req DTO.CreateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	if strings.TrimSpace(req.FirstName) == "" || strings.TrimSpace(req.LastName) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "first_name and last_name are required"})
		return
	}

	cl := domain.Client{
		ID:        uuid.New(),
		UserID:    userID,
		FirstName: strings.TrimSpace(req.FirstName),
		LastName:  strings.TrimSpace(req.LastName),
		Height:    req.Height,
		Weight:    req.Weight,
		Goal:      req.Goal,
	}

	if err := a.db.Create(&cl).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create client"})
		return
	}

	c.JSON(http.StatusCreated, clientToResponse(cl, clientSessionStats{}))
}

// handleGetClients returns clients owned by the current trainer.
// @Summary List clients
// @Tags clients
// @Produce json
// @Security BearerAuth
// @Success 200 {array} DTO.ClientResponse
// @Failure 401 {object} DTO.ErrorResponse
// @Failure 500 {object} DTO.ErrorResponse
// @Router /api/v1/clients [get]
func (a *App) handleGetClients(c *gin.Context) {
	userID := uuid.MustParse(c.MustGet("user_id").(string))

	var clientsDB []domain.Client
	if err := a.db.Where("user_id = ?", userID).Order("last_name, first_name").Find(&clientsDB).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load clients"})
		return
	}

	clientIDs := make([]uuid.UUID, 0, len(clientsDB))
	for _, cl := range clientsDB {
		clientIDs = append(clientIDs, cl.ID)
	}

	stats, err := a.loadClientSessionStats(userID, clientIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load client sessions"})
		return
	}

	resp := make([]DTO.ClientResponse, 0, len(clientsDB))
	for _, cl := range clientsDB {
		resp = append(resp, clientToResponse(cl, stats[cl.ID]))
	}

	c.JSON(http.StatusOK, resp)
}

// handleGetClientByID returns one client owned by the current trainer.
// @Summary Get client
// @Tags clients
// @Produce json
// @Security BearerAuth
// @Param id path string true "Client ID"
// @Success 200 {object} DTO.ClientResponse
// @Failure 400 {object} DTO.ErrorResponse
// @Failure 401 {object} DTO.ErrorResponse
// @Failure 404 {object} DTO.ErrorResponse
// @Failure 500 {object} DTO.ErrorResponse
// @Router /api/v1/clients/{id} [get]
func (a *App) handleGetClientByID(c *gin.Context) {
	userID := uuid.MustParse(c.MustGet("user_id").(string))

	clientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client id"})
		return
	}

	cl, ok := a.findClientForTrainer(c, userID, clientID)
	if !ok {
		return
	}

	stats, err := a.loadClientSessionStats(userID, []uuid.UUID{clientID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load client sessions"})
		return
	}

	c.JSON(http.StatusOK, clientToResponse(cl, stats[clientID]))
}

// handleUpdateClient partially updates one client owned by the current trainer.
// @Summary Update client
// @Tags clients
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Client ID"
// @Param request body DTO.UpdateClientRequest true "Client data"
// @Success 200 {object} DTO.ClientResponse
// @Failure 400 {object} DTO.ErrorResponse
// @Failure 401 {object} DTO.ErrorResponse
// @Failure 404 {object} DTO.ErrorResponse
// @Failure 500 {object} DTO.ErrorResponse
// @Router /api/v1/clients/{id} [patch]
func (a *App) handleUpdateClient(c *gin.Context) {
	userID := uuid.MustParse(c.MustGet("user_id").(string))

	clientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client id"})
		return
	}

	var req DTO.UpdateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	updates := make(map[string]interface{})
	if req.FirstName != nil {
		firstName := strings.TrimSpace(*req.FirstName)
		if firstName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "first_name cannot be empty"})
			return
		}
		updates["first_name"] = firstName
	}
	if req.LastName != nil {
		lastName := strings.TrimSpace(*req.LastName)
		if lastName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "last_name cannot be empty"})
			return
		}
		updates["last_name"] = lastName
	}
	if req.Height != nil {
		updates["height"] = *req.Height
	}
	if req.Weight != nil {
		updates["weight"] = *req.Weight
	}
	if req.Goal != nil {
		updates["goal"] = *req.Goal
	}

	var cl domain.Client
	err = a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND user_id = ?", clientID, userID).First(&cl).Error; err != nil {
			return err
		}
		if len(updates) == 0 {
			return nil
		}
		if err := tx.Model(&cl).Updates(updates).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND user_id = ?", clientID, userID).First(&cl).Error
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update client"})
		return
	}

	stats, err := a.loadClientSessionStats(userID, []uuid.UUID{clientID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load client sessions"})
		return
	}

	c.JSON(http.StatusOK, clientToResponse(cl, stats[clientID]))
}

// handleDeleteClient deletes one client owned by the current trainer.
// @Summary Delete client
// @Tags clients
// @Security BearerAuth
// @Param id path string true "Client ID"
// @Success 204
// @Failure 400 {object} DTO.ErrorResponse
// @Failure 401 {object} DTO.ErrorResponse
// @Failure 404 {object} DTO.ErrorResponse
// @Failure 500 {object} DTO.ErrorResponse
// @Router /api/v1/clients/{id} [delete]
func (a *App) handleDeleteClient(c *gin.Context) {
	userID := uuid.MustParse(c.MustGet("user_id").(string))

	clientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client id"})
		return
	}

	err = a.db.Transaction(func(tx *gorm.DB) error {
		var cl domain.Client
		if err := tx.Where("id = ? AND user_id = ?", clientID, userID).First(&cl).Error; err != nil {
			return err
		}
		if err := tx.Where("client_id = ?", clientID).Delete(&domain.WorkoutClient{}).Error; err != nil {
			return err
		}
		if err := tx.Where("client_id = ? AND trainer_id = ?", clientID, userID).Delete(&domain.WorkoutPackage{}).Error; err != nil {
			return err
		}
		return tx.Delete(&cl).Error
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete client"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (a *App) findClientForTrainer(c *gin.Context, userID, clientID uuid.UUID) (domain.Client, bool) {
	var cl domain.Client
	if err := a.db.Where("id = ? AND user_id = ?", clientID, userID).First(&cl).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
			return domain.Client{}, false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load client"})
		return domain.Client{}, false
	}
	return cl, true
}

func (a *App) loadClientSessionStats(userID uuid.UUID, clientIDs []uuid.UUID) (map[uuid.UUID]clientSessionStats, error) {
	stats := make(map[uuid.UUID]clientSessionStats, len(clientIDs))
	if len(clientIDs) == 0 {
		return stats, nil
	}

	for _, clientID := range clientIDs {
		stats[clientID] = clientSessionStats{ClientID: clientID}
	}

	var totals []clientSessionStats
	if err := a.db.Table("workout_clients").
		Select("workout_clients.client_id, COUNT(*) AS total_sessions").
		Joins("JOIN workouts ON workouts.id = workout_clients.workout_id").
		Where("workouts.user_id = ? AND workout_clients.client_id IN ?", userID, clientIDs).
		Group("workout_clients.client_id").
		Scan(&totals).Error; err != nil {
		return nil, err
	}
	for _, row := range totals {
		stat := stats[row.ClientID]
		stat.TotalSessions = row.TotalSessions
		stats[row.ClientID] = stat
	}

	var left []clientSessionStats
	if err := a.db.Model(&domain.WorkoutPackage{}).
		Select("client_id, COALESCE(SUM(total_count - used_count), 0) AS left_sessions").
		Where("trainer_id = ? AND is_active = ? AND client_id IN ?", userID, true, clientIDs).
		Group("client_id").
		Scan(&left).Error; err != nil {
		return nil, err
	}
	for _, row := range left {
		stat := stats[row.ClientID]
		stat.LeftSessions = row.LeftSessions
		stats[row.ClientID] = stat
	}

	return stats, nil
}

func clientToResponse(cl domain.Client, stats clientSessionStats) DTO.ClientResponse {
	return DTO.ClientResponse{
		ID:            cl.ID.String(),
		UserID:        cl.UserID.String(),
		FirstName:     cl.FirstName,
		LastName:      cl.LastName,
		Height:        cl.Height,
		Weight:        cl.Weight,
		Goal:          cl.Goal,
		TotalSessions: stats.TotalSessions,
		LeftSessions:  stats.LeftSessions,
	}
}
