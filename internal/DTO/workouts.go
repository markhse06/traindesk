package DTO

type CreateWorkoutRequest struct {
	DateTime    string   `json:"date_time"`
	DurationMin int      `json:"duration_min"`
	Price       float64  `json:"price"`
	Type        string   `json:"type"`
	Status      string   `json:"status"`
	ClientIDs   []string `json:"client_ids"`
	Notes       string   `json:"notes"`
	PackageID   *string  `json:"package_id,omitempty"`
	UpdatedAt   *string  `json:"updated_at,omitempty"`
}

type WorkoutResponse struct {
	ID          string   `json:"id"`
	DateTime    string   `json:"datetime"`
	DurationMin int      `json:"duration_min"`
	Price       float64  `json:"price"`
	Type        string   `json:"type"`
	Status      string   `json:"status"`
	ClientIDs   []string `json:"client_ids"`
	PackageID   *string  `json:"package_id,omitempty"`
	Notes       string   `json:"notes"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type UpdateWorkoutRequest struct {
	DateTime    *string   `json:"date_time"`
	DurationMin *int      `json:"duration_min"`
	Price       *float64  `json:"price"`
	Type        *string   `json:"type"`
	Status      *string   `json:"status"`
	ClientIDs   *[]string `json:"client_ids"`
	Notes       *string   `json:"notes"`
	UpdatedAt   *string   `json:"updated_at,omitempty"`
}
