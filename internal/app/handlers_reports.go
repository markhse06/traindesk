package app

import (
	"net/http"
	"time"
	"traindesk/internal/DTO"
	"traindesk/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type reportTotalsRow struct {
	TotalWorkouts    int
	TotalDurationMin int
	TotalRevenue     float64
}

type reportActiveClientsRow struct {
	ActiveClients int
}

type reportTypeRow struct {
	Type         string
	Count        int
	TotalRevenue float64
}

type reportClientRow struct {
	ClientID      uuid.UUID
	FirstName     string
	LastName      string
	WorkoutCount  int
	TotalDuration int
}

// handleGetReportSummary returns financial and activity statistics for a period.
// @Summary Report summary
// @Tags reports
// @Produce json
// @Security BearerAuth
// @Param started_at query string false "Period start, RFC3339"
// @Param ended_at query string false "Period end, RFC3339"
// @Success 200 {object} DTO.ReportSummaryResponse
// @Failure 400 {object} DTO.ErrorResponse
// @Failure 401 {object} DTO.ErrorResponse
// @Failure 500 {object} DTO.ErrorResponse
// @Router /api/v1/reports/summary [get]
func (a *App) handleGetReportSummary(c *gin.Context) {
	userID := uuid.MustParse(c.MustGet("user_id").(string))

	startedAt, endedAt, ok := parseReportPeriod(c)
	if !ok {
		return
	}

	base := a.db.Model(&domain.Workout{}).Where("user_id = ?", userID)
	if startedAt != nil {
		base = base.Where("date_time >= ?", *startedAt)
	}
	if endedAt != nil {
		base = base.Where("date_time <= ?", *endedAt)
	}

	var totals reportTotalsRow
	if err := base.Select(
		"COUNT(*) AS total_workouts, COALESCE(SUM(duration_min), 0) AS total_duration_min, COALESCE(SUM(price), 0) AS total_revenue",
	).Scan(&totals).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build report"})
		return
	}

	var active reportActiveClientsRow
	activeQuery := a.db.Table("workouts").
		Joins("JOIN workout_clients ON workout_clients.workout_id = workouts.id").
		Where("workouts.user_id = ?", userID)
	if startedAt != nil {
		activeQuery = activeQuery.Where("workouts.date_time >= ?", *startedAt)
	}
	if endedAt != nil {
		activeQuery = activeQuery.Where("workouts.date_time <= ?", *endedAt)
	}
	if err := activeQuery.Select("COUNT(DISTINCT workout_clients.client_id) AS active_clients").Scan(&active).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build report"})
		return
	}

	byType, err := a.loadReportByType(userID, startedAt, endedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build report"})
		return
	}

	byClient, err := a.loadReportByClient(userID, startedAt, endedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build report"})
		return
	}

	c.JSON(http.StatusOK, DTO.ReportSummaryResponse{
		StartedAt:        formatOptionalReportTime(startedAt),
		EndedAt:          formatOptionalReportTime(endedAt),
		TotalWorkouts:    totals.TotalWorkouts,
		TotalDurationMin: totals.TotalDurationMin,
		TotalRevenue:     totals.TotalRevenue,
		ActiveClients:    active.ActiveClients,
		ByType:           byType,
		ByClient:         byClient,
	})
}

func parseReportPeriod(c *gin.Context) (*time.Time, *time.Time, bool) {
	startedAt, err := parseOptionalRFC3339(stringPtrIfNotEmpty(c.Query("started_at")))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid started_at format, use RFC3339"})
		return nil, nil, false
	}
	endedAt, err := parseOptionalRFC3339(stringPtrIfNotEmpty(c.Query("ended_at")))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ended_at format, use RFC3339"})
		return nil, nil, false
	}
	if startedAt != nil && endedAt != nil && startedAt.After(*endedAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "started_at must be before ended_at"})
		return nil, nil, false
	}
	return startedAt, endedAt, true
}

func (a *App) loadReportByType(userID uuid.UUID, startedAt, endedAt *time.Time) ([]DTO.ReportWorkoutTypeSummary, error) {
	query := a.db.Model(&domain.Workout{}).
		Select("type, COUNT(*) AS count, COALESCE(SUM(price), 0) AS total_revenue").
		Where("user_id = ?", userID).
		Group("type").
		Order("type")
	if startedAt != nil {
		query = query.Where("date_time >= ?", *startedAt)
	}
	if endedAt != nil {
		query = query.Where("date_time <= ?", *endedAt)
	}

	var rows []reportTypeRow
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]DTO.ReportWorkoutTypeSummary, 0, len(rows))
	for _, row := range rows {
		result = append(result, DTO.ReportWorkoutTypeSummary{
			Type:         row.Type,
			Count:        row.Count,
			TotalRevenue: row.TotalRevenue,
		})
	}
	return result, nil
}

func (a *App) loadReportByClient(userID uuid.UUID, startedAt, endedAt *time.Time) ([]DTO.ReportClientActivity, error) {
	query := a.db.Table("workout_clients").
		Select("clients.id AS client_id, clients.first_name, clients.last_name, COUNT(*) AS workout_count, COALESCE(SUM(workouts.duration_min), 0) AS total_duration").
		Joins("JOIN workouts ON workouts.id = workout_clients.workout_id").
		Joins("JOIN clients ON clients.id = workout_clients.client_id").
		Where("workouts.user_id = ? AND clients.user_id = ?", userID, userID).
		Group("clients.id, clients.first_name, clients.last_name").
		Order("workout_count DESC, clients.last_name, clients.first_name")
	if startedAt != nil {
		query = query.Where("workouts.date_time >= ?", *startedAt)
	}
	if endedAt != nil {
		query = query.Where("workouts.date_time <= ?", *endedAt)
	}

	var rows []reportClientRow
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]DTO.ReportClientActivity, 0, len(rows))
	for _, row := range rows {
		result = append(result, DTO.ReportClientActivity{
			ClientID:      row.ClientID.String(),
			FirstName:     row.FirstName,
			LastName:      row.LastName,
			WorkoutCount:  row.WorkoutCount,
			TotalDuration: row.TotalDuration,
		})
	}
	return result, nil
}

func stringPtrIfNotEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func formatOptionalReportTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return formatAPITime(*value)
}
