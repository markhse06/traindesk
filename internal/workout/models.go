package workout

// CreateWorkoutRequest — тело запроса при создании тренировки.
type CreateWorkoutRequest struct {
	Date        string   `json:"date"`         // YYYY-MM-DD
	DurationMin int      `json:"duration_min"` // 1–300
	Type        string   `json:"type"`         // "cardio", "strength", "stretch", "functional"
	ClientIDs   []string `json:"client_ids"`   // 0, 1 или несколько клиентов
	Notes       string   `json:"notes"`
}

// WorkoutResponse — то, что отдаём клиенту.
type WorkoutResponse struct {
	ID          string   `json:"id"`
	Date        string   `json:"date"`
	DurationMin int      `json:"duration_min"`
	Type        string   `json:"type"`
	ClientIDs   []string `json:"client_ids"`
	Notes       string   `json:"notes"`
}
