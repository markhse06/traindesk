package app

import (
	"net/http"
	"traindesk/internal/DTO"
	"traindesk/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// handleCreatePackage creates a workout package for a trainer's client.
// @Summary Create workout package
// @Tags workout-packages
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body DTO.CreatePackageRequest true "Workout package data"
// @Success 201 {object} domain.WorkoutPackage
// @Failure 400 {object} DTO.ErrorResponse
// @Failure 401 {object} DTO.ErrorResponse
// @Failure 500 {object} DTO.ErrorResponse
// @Router /api/v1/packages [post]
func (a *App) handleCreatePackage(c *gin.Context) {
	trainerID := uuid.MustParse(c.MustGet("user_id").(string))

	var req DTO.CreatePackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	clientUUID, err := uuid.Parse(req.ClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client_id"})
		return
	}
	updatedAt, err := parseOptionalRFC3339(req.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid updated_at format, use RFC3339"})
		return
	}
	if _, ok := a.findClientForTrainer(c, trainerID, clientUUID); !ok {
		return
	}

	pkg := domain.WorkoutPackage{
		ID:         uuid.New(),
		TrainerID:  trainerID,
		ClientID:   clientUUID,
		TotalCount: req.TotalCount,
		Price:      req.Price,
		IsActive:   true,
	}
	if updatedAt != nil {
		pkg.UpdatedAt = *updatedAt
	}

	if err := a.db.Create(&pkg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create package"})
		return
	}

	c.JSON(http.StatusCreated, packageToResponse(pkg))
}

func (a *App) handleGetClientPackages(c *gin.Context) {
	trainerID := c.MustGet("user_id").(string)
	clientID := c.Param("client_id")

	var packages []domain.WorkoutPackage
	err := a.db.Where("trainer_id = ? AND client_id = ? AND is_active = ?", trainerID, clientID, true).
		Find(&packages).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch packages"})
		return
	}

	resp := make([]DTO.WorkoutPackageResponse, 0, len(packages))
	for _, pkg := range packages {
		resp = append(resp, packageToResponse(pkg))
	}

	c.JSON(http.StatusOK, resp)
}

func packageToResponse(pkg domain.WorkoutPackage) DTO.WorkoutPackageResponse {
	return DTO.WorkoutPackageResponse{
		ID:         pkg.ID.String(),
		TrainerID:  pkg.TrainerID.String(),
		ClientID:   pkg.ClientID.String(),
		TotalCount: pkg.TotalCount,
		UsedCount:  pkg.UsedCount,
		IsActive:   pkg.IsActive,
		Price:      pkg.Price,
		CreatedAt:  formatAPITime(pkg.CreatedAt),
		UpdatedAt:  formatAPITime(pkg.UpdatedAt),
	}
}
