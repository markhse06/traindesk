package app

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"traindesk/internal/client"
	"traindesk/internal/workout"
)

// handleCreateWorkout — создать тренировку (индивидуальную или групповую).
func (a *App) handleCreateWorkout(c *gin.Context) {
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

	var req workout.CreateWorkoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	if req.Date == "" || req.DurationMin < 1 || req.DurationMin > 300 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "date and duration_min (1-300) are required",
		})
		return
	}

	if !workout.IsValidType(req.Type) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "invalid workout type",
			"allowed_types": workout.ValidWorkoutTypes,
		})
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid date format, expected YYYY-MM-DD",
		})
		return
	}

	// Парсим client_ids в UUID и проверяем, что все клиенты принадлежат текущему тренеру.
	clientUUIDs := make([]uuid.UUID, 0, len(req.ClientIDs))
	for _, cidStr := range req.ClientIDs {
		cid, err := uuid.Parse(cidStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client_id: " + cidStr})
			return
		}
		clientUUIDs = append(clientUUIDs, cid)
	}

	if len(clientUUIDs) > 0 {
		var cnt int64
		if err := a.db.
			Model(&client.Client{}).
			Where("user_id = ? AND id IN ?", userID, clientUUIDs).
			Count(&cnt).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate clients"})
			return
		}
		if cnt != int64(len(clientUUIDs)) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "one or more client_ids do not belong to the current user",
			})
			return
		}
	}

	w := workout.Workout{
		ID:          uuid.New(),
		UserID:      userID,
		Date:        date,
		DurationMin: req.DurationMin,
		Type:        workout.WorkoutType(req.Type),
		Notes:       req.Notes,
	}

	err = a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&w).Error; err != nil {
			return err
		}

		if len(clientUUIDs) > 0 {
			links := make([]workout.WorkoutClient, 0, len(clientUUIDs))
			for _, cid := range clientUUIDs {
				links = append(links, workout.WorkoutClient{
					WorkoutID: w.ID,
					ClientID:  cid,
				})
			}
			if err := tx.Create(&links).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create workout"})
		return
	}

	resp := workout.WorkoutResponse{
		ID:          w.ID.String(),
		Date:        w.Date.Format("2006-01-02"),
		DurationMin: w.DurationMin,
		Type:        string(w.Type),
		ClientIDs:   req.ClientIDs,
		Notes:       w.Notes,
	}

	c.JSON(http.StatusCreated, resp)
}

// handleGetWorkouts — список тренировок текущего тренера с client_ids.
func (a *App) handleGetWorkouts(c *gin.Context) {
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

	var workoutsDB []workout.Workout
	if err := a.db.Where("user_id = ?", userID).Order("date desc").Find(&workoutsDB).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load workouts"})
		return
	}

	workoutIDs := make([]uuid.UUID, 0, len(workoutsDB))
	for _, w := range workoutsDB {
		workoutIDs = append(workoutIDs, w.ID)
	}

	linksMap := make(map[uuid.UUID][]string)
	if len(workoutIDs) > 0 {
		var links []workout.WorkoutClient
		if err := a.db.Where("workout_id IN ?", workoutIDs).Find(&links).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load workout clients"})
			return
		}

		for _, l := range links {
			linksMap[l.WorkoutID] = append(linksMap[l.WorkoutID], l.ClientID.String())
		}
	}

	resp := make([]workout.WorkoutResponse, 0, len(workoutsDB))
	for _, w := range workoutsDB {
		resp = append(resp, workout.WorkoutResponse{
			ID:          w.ID.String(),
			Date:        w.Date.Format("2006-01-02"),
			DurationMin: w.DurationMin,
			Type:        string(w.Type),
			ClientIDs:   linksMap[w.ID], // это []string
			Notes:       w.Notes,
		})
	}

	c.JSON(http.StatusOK, resp)
}

func (a *App) handleGetWorkoutByID(c *gin.Context) {
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

	workoutIDStr := c.Param("id")
	workoutID, err := uuid.Parse(workoutIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workout id"})
		return
	}

	var w workout.Workout
	if err := a.db.Where("id = ? AND user_id = ?", workoutID, userID).First(&w).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "workout not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load workout"})
		}
		return
	}

	// Подтягиваем связанных клиентов.
	var links []workout.WorkoutClient
	if err := a.db.Where("workout_id = ?", w.ID).Find(&links).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load workout clients"})
		return
	}

	clientIDs := make([]string, 0, len(links))
	for _, l := range links {
		clientIDs = append(clientIDs, l.ClientID.String())
	}

	resp := workout.WorkoutResponse{
		ID:          w.ID.String(),
		Date:        w.Date.Format("2006-01-02"),
		DurationMin: w.DurationMin,
		Type:        string(w.Type),
		ClientIDs:   clientIDs,
		Notes:       w.Notes,
	}

	c.JSON(http.StatusOK, resp)
}

func (a *App) handleUpdateWorkout(c *gin.Context) {
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

	workoutIDStr := c.Param("id")
	workoutID, err := uuid.Parse(workoutIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workout id"})
		return
	}

	var existing workout.Workout
	if err := a.db.Where("id = ? AND user_id = ?", workoutID, userID).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "workout not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load workout"})
		}
		return
	}

	var req workout.CreateWorkoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	if req.Date == "" || req.DurationMin < 1 || req.DurationMin > 300 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "date and duration_min (1-300) are required",
		})
		return
	}

	if !workout.IsValidType(req.Type) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "invalid workout type",
			"allowed_types": workout.ValidWorkoutTypes,
		})
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid date format, expected YYYY-MM-DD",
		})
		return
	}

	// Разбираем client_ids и проверяем их владельца.
	clientUUIDs := make([]uuid.UUID, 0, len(req.ClientIDs))
	for _, cidStr := range req.ClientIDs {
		cid, err := uuid.Parse(cidStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client_id: " + cidStr})
			return
		}
		clientUUIDs = append(clientUUIDs, cid)
	}

	if len(clientUUIDs) > 0 {
		var cnt int64
		if err := a.db.
			Model(&client.Client{}).
			Where("user_id = ? AND id IN ?", userID, clientUUIDs).
			Count(&cnt).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate clients"})
			return
		}
		if cnt != int64(len(clientUUIDs)) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "one or more client_ids do not belong to the current user",
			})
			return
		}
	}

	existing.Date = date
	existing.DurationMin = req.DurationMin
	existing.Type = workout.WorkoutType(req.Type)
	existing.Notes = req.Notes

	err = a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&existing).Error; err != nil {
			return err
		}

		// Сначала удаляем старые связи.
		if err := tx.Where("workout_id = ?", existing.ID).Delete(&workout.WorkoutClient{}).Error; err != nil {
			return err
		}

		// Затем добавляем новые связи.
		if len(clientUUIDs) > 0 {
			links := make([]workout.WorkoutClient, 0, len(clientUUIDs))
			for _, cid := range clientUUIDs {
				links = append(links, workout.WorkoutClient{
					WorkoutID: existing.ID,
					ClientID:  cid,
				})
			}
			if err := tx.Create(&links).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update workout"})
		return
	}

	resp := workout.WorkoutResponse{
		ID:          existing.ID.String(),
		Date:        existing.Date.Format("2006-01-02"),
		DurationMin: existing.DurationMin,
		Type:        string(existing.Type),
		ClientIDs:   req.ClientIDs,
		Notes:       existing.Notes,
	}

	c.JSON(http.StatusOK, resp)
}

func (a *App) handleDeleteWorkout(c *gin.Context) {
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

	workoutIDStr := c.Param("id")
	workoutID, err := uuid.Parse(workoutIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workout id"})
		return
	}

	// Проверяем, что тренировка принадлежит пользователю.
	var w workout.Workout
	if err := a.db.Where("id = ? AND user_id = ?", workoutID, userID).First(&w).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "workout not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load workout"})
		}
		return
	}

	err = a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("workout_id = ?", w.ID).Delete(&workout.WorkoutClient{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&w).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete workout"})
		return
	}

	c.Status(http.StatusNoContent)
}
