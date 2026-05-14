package DTO

type CreatePackageRequest struct {
	ClientID   string  `json:"client_id" binding:"required"`
	TotalCount int     `json:"total_count" binding:"required"`
	Price      float64 `json:"price"`
}
