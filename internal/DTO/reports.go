package DTO

type ReportClientActivity struct {
	ClientID      string `json:"client_id"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	WorkoutCount  int    `json:"workout_count"`
	TotalDuration int    `json:"total_duration_min"`
}

type ReportWorkoutTypeSummary struct {
	Type         string  `json:"type"`
	Count        int     `json:"count"`
	TotalRevenue float64 `json:"total_revenue"`
}

type ReportSummaryResponse struct {
	StartedAt        string                     `json:"started_at"`
	EndedAt          string                     `json:"ended_at"`
	TotalWorkouts    int                        `json:"total_workouts"`
	TotalDurationMin int                        `json:"total_duration_min"`
	TotalRevenue     float64                    `json:"total_revenue"`
	ActiveClients    int                        `json:"active_clients"`
	ByType           []ReportWorkoutTypeSummary `json:"by_type"`
	ByClient         []ReportClientActivity     `json:"by_client"`
}
