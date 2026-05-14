package DTO

// CreateWorkoutRequest — тело запроса при создании тренировки.
type CreateWorkoutRequest struct {
	DateTime    string   `json:"date_time"`    // YYYY-MM-DD
	DurationMin int      `json:"duration_min"` // 1–300
	Price       float64  `json:"price"`
	Type        string   `json:"type"`       // "cardio", "strength", "stretch", "functional"
	ClientIDs   []string `json:"client_ids"` // 0, 1 или несколько клиентов
	Notes       string   `json:"notes"`
	PackageID   *string  `json:"package_id,omitempty"`
}

// WorkoutResponse — то, что отдаём клиенту.
type WorkoutResponse struct {
	ID          string   `json:"id"`
	DateTime    string   `json:"datetime"`
	DurationMin int      `json:"duration_min"`
	Price       float64  `json:"price"`
	Type        string   `json:"type"`
	ClientIDs   []string `json:"client_ids"`
	Notes       string   `json:"notes"`
}

type UpdateWorkoutRequest struct {
	DateTime    *string   `json:"date_time"`    // Указатель
	DurationMin *int      `json:"duration_min"` // Указатель
	Price       *float64  `json:"price"`
	Type        *string   `json:"type"`       // Указатель
	ClientIDs   *[]string `json:"client_ids"` // Указатель на слайс
	Notes       *string   `json:"notes"`      // Указатель
}
