package app

import (
	"fmt"
	"net/http"
	"time"
	"traindesk/internal/DTO"
	"traindesk/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// handleCreateWorkout — создать тренировку (индивидуальную или групповую).
// @Summary Create workout
// @Tags workouts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body DTO.CreateWorkoutRequest true "Workout data"
// @Success 201 {object} DTO.WorkoutResponse
// @Failure 400 {object} DTO.ErrorResponse
// @Failure 401 {object} DTO.ErrorResponse
// @Failure 500 {object} DTO.ErrorResponse
// @Router /api/v1/workouts [post]
func (a *App) handleCreateWorkout(c *gin.Context) {
	userID := c.MustGet("user_id").(string) // Используем MustGet, так как Middleware гарантирует наличие
	userUUID, _ := uuid.Parse(userID)

	var req DTO.CreateWorkoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	if len(req.ClientIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one client is required for a workout"})
		return
	}

	// Валидация времени
	dateTime, err := time.Parse(time.RFC3339, req.DateTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, use RFC3339 (e.g. 2026-05-12T15:00:00Z)", "details": err.Error()})
		return
	}

	clientUUIDs := make([]uuid.UUID, 0, len(req.ClientIDs))
	for _, cidStr := range req.ClientIDs {
		cid, _ := uuid.Parse(cidStr)
		clientUUIDs = append(clientUUIDs, cid)
	}

	// Создаем объект тренировки
	w := domain.Workout{
		ID:          uuid.New(),
		UserID:      userUUID,
		DateTime:    dateTime,
		DurationMin: req.DurationMin,
		Type:        domain.WorkoutType(req.Type),
		Notes:       req.Notes,
	}

	err = a.db.Transaction(func(tx *gorm.DB) error {
		// Проверяем клиентов
		var clients []domain.Client
		tx.Where("user_id = ? AND id IN ?", userUUID, clientUUIDs).Find(&clients)
		if len(clients) != len(clientUUIDs) {
			return fmt.Errorf("invalid clients")
		}

		// Сохраняем тренировку
		if err := tx.Create(&w).Error; err != nil {
			return err
		}

		// Создаем связи в many2many таблице
		if err := tx.Model(&w).Association("Clients").Replace(clients); err != nil {
			return err
		}

		if req.PackageID != nil && *req.PackageID != "" {
			pkgID, err := uuid.Parse(*req.PackageID)
			if err != nil {
				return err
			}
			var pkg domain.WorkoutPackage

			// Находим пакет и проверяем, есть ли в нем место
			if err := tx.Where("id = ? AND trainer_id = ? AND used_count < total_count", pkgID, userUUID).First(&pkg).Error; err != nil {
				return fmt.Errorf("active package not found or no sessions left")
			}

			// Привязываем тренировку к пакету (нужно добавить поле PackageID в domain.Workout)
			w.PackageID = pkgID
			if err := tx.Save(&w).Error; err != nil {
				return err
			}

			// Инкрементируем счетчик
			newUsed := pkg.UsedCount + 1
			updates := map[string]interface{}{"used_count": newUsed}
			if newUsed >= pkg.TotalCount {
				updates["is_active"] = false
			}

			if err := tx.Model(&pkg).Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, w)
}

// handleGetWorkouts — список тренировок текущего тренера с client_ids.
// @Summary List workouts
// @Tags workouts
// @Produce json
// @Security BearerAuth
// @Success 200 {array} DTO.WorkoutResponse
// @Failure 401 {object} DTO.ErrorResponse
// @Failure 500 {object} DTO.ErrorResponse
// @Router /api/v1/workouts [get]
func (a *App) handleGetWorkouts(c *gin.Context) {
	// Получаем ID пользователя из контекста (установлен Middleware)
	userID := c.MustGet("user_id").(string)
	userUUID, _ := uuid.Parse(userID)

	// Получаем фильтры из Query параметров
	startedAtStr := c.Query("started_at")
	endedAtStr := c.Query("ended_at")

	// Строим запрос через GORM
	query := a.db.Preload("Clients").Where("user_id = ?", userUUID)

	// Добавляем фильтрацию, если параметры переданы
	if startedAtStr != "" {
		t, err := time.Parse(time.RFC3339, startedAtStr)
		if err == nil {
			query = query.Where("date_time >= ?", t)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid started_at format, use RFC3339"})
			return
		}
	}

	if endedAtStr != "" {
		t, err := time.Parse(time.RFC3339, endedAtStr)
		if err == nil {
			query = query.Where("date_time <= ?", t)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ended_at format, use RFC3339"})
			return
		}
	}

	var workoutsDB []domain.Workout
	if err := query.Order("date_time asc").Find(&workoutsDB).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load workouts"})
		return
	}

	// Мапим результат в DTO
	// Благодаря Preload("Clients"), данные о клиентах уже лежат внутри workoutsDB[i].Clients
	resp := make([]DTO.WorkoutResponse, 0, len(workoutsDB))
	for _, w := range workoutsDB {
		clientIDs := make([]string, 0, len(w.Clients))
		for _, cl := range w.Clients {
			clientIDs = append(clientIDs, cl.ID.String())
		}

		resp = append(resp, DTO.WorkoutResponse{
			ID:          w.ID.String(),
			DateTime:    w.DateTime.Format(time.RFC3339),
			DurationMin: w.DurationMin,
			Type:        string(w.Type),
			ClientIDs:   clientIDs,
			Notes:       w.Notes,
		})
	}

	c.JSON(http.StatusOK, resp)
}

// @Summary Get workout
// @Tags workouts
// @Produce json
// @Security BearerAuth
// @Param id path string true "Workout ID"
// @Success 200 {object} DTO.WorkoutResponse
// @Failure 400 {object} DTO.ErrorResponse
// @Failure 401 {object} DTO.ErrorResponse
// @Failure 404 {object} DTO.ErrorResponse
// @Failure 500 {object} DTO.ErrorResponse
// @Router /api/v1/workouts/{id} [get]
func (a *App) handleGetWorkoutByID(c *gin.Context) {
	userIDStr := c.MustGet("user_id").(string)

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

	var w domain.Workout
	if err := a.db.Where("id = ? AND user_id = ?", workoutID, userID).First(&w).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "workout not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load workout"})
		}
		return
	}

	// Подтягиваем связанных клиентов.
	var links []domain.WorkoutClient
	if err := a.db.Where("workout_id = ?", w.ID).Find(&links).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load workout clients"})
		return
	}

	clientIDs := make([]string, 0, len(links))
	for _, l := range links {
		clientIDs = append(clientIDs, l.ClientID.String())
	}

	resp := DTO.WorkoutResponse{
		ID:          w.ID.String(),
		DateTime:    w.DateTime.Format(time.RFC3339),
		DurationMin: w.DurationMin,
		Type:        string(w.Type),
		ClientIDs:   clientIDs,
		Notes:       w.Notes,
	}

	c.JSON(http.StatusOK, resp)
}

// @Summary Update workout
// @Tags workouts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Workout ID"
// @Param request body DTO.CreateWorkoutRequest true "Workout data"
// @Success 200 {object} DTO.WorkoutResponse
// @Failure 400 {object} DTO.ErrorResponse
// @Failure 401 {object} DTO.ErrorResponse
// @Failure 404 {object} DTO.ErrorResponse
// @Failure 500 {object} DTO.ErrorResponse
// @Router /api/v1/workouts/{id} [put]
func (a *App) handleUpdateWorkout(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	userUUID, _ := uuid.Parse(userID)

	workoutID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workout id"})
		return
	}

	// Находим существующую тренировку
	var existing domain.Workout
	if err := a.db.Where("id = ? AND user_id = ?", workoutID, userUUID).First(&existing).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "workout not found"})
		return
	}

	// Десериализуем в Update структуру
	var req DTO.UpdateWorkoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	// Частичное обновление полей
	if req.DateTime != nil {
		dateTime, err := time.Parse(time.RFC3339, *req.DateTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
			return
		}
		existing.DateTime = dateTime // Предположим, поле в БД называется Date
	}

	if req.DurationMin != nil {
		if *req.DurationMin < 1 || *req.DurationMin > 300 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "duration_min must be 1-300"})
			return
		}
		existing.DurationMin = *req.DurationMin
	}

	if req.Type != nil {
		if !domain.IsValidType(*req.Type) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workout type"})
			return
		}
		existing.Type = domain.WorkoutType(*req.Type)
	}

	if req.Notes != nil {
		existing.Notes = *req.Notes
	}

	// Логика обновления клиентов (если переданы)
	err = a.db.Transaction(func(tx *gorm.DB) error {
		// Сохраняем основные поля
		if err := tx.Save(&existing).Error; err != nil {
			return err
		}

		// Если client_ids переданы явно (даже если это пустой список)
		if req.ClientIDs != nil {
			// Валидация и обновление связей many2many
			clientUUIDs := make([]uuid.UUID, 0, len(*req.ClientIDs))
			for _, cidStr := range *req.ClientIDs {
				cid, _ := uuid.Parse(cidStr)
				clientUUIDs = append(clientUUIDs, cid)
			}

			var clients []domain.Client
			tx.Where("user_id = ? AND id IN ?", userUUID, clientUUIDs).Find(&clients)
			if len(clients) != len(clientUUIDs) {
				return fmt.Errorf("one or more client_ids are invalid")
			}

			// GORM Association Replace удалит старые связи и запишет новые
			if err := tx.Model(&existing).Association("Clients").Replace(clients); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, existing)
}

// @Summary Delete workout
// @Tags workouts
// @Security BearerAuth
// @Param id path string true "Workout ID"
// @Success 204
// @Failure 400 {object} DTO.ErrorResponse
// @Failure 401 {object} DTO.ErrorResponse
// @Failure 404 {object} DTO.ErrorResponse
// @Failure 500 {object} DTO.ErrorResponse
// @Router /api/v1/workouts/{id} [delete]
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
	var w domain.Workout
	if err := a.db.Where("id = ? AND user_id = ?", workoutID, userID).First(&w).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "workout not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load workout"})
		}
		return
	}

	err = a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("workout_id = ?", w.ID).Delete(&domain.WorkoutClient{}).Error; err != nil {
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
