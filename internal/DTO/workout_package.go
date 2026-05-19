package DTO

type CreatePackageRequest struct {
	ClientID   string  `json:"client_id" binding:"required"`
	TotalCount int     `json:"total_count" binding:"required"`
	Price      float64 `json:"price"`
	UpdatedAt  *string `json:"updated_at,omitempty"`
}

type WorkoutPackageResponse struct {
	ID         string  `json:"id"`
	TrainerID  string  `json:"trainer_id"`
	ClientID   string  `json:"client_id"`
	TotalCount int     `json:"total_count"`
	UsedCount  int     `json:"used_count"`
	IsActive   bool    `json:"is_active"`
	Price      float64 `json:"price"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}
