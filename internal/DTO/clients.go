package DTO

// CreateClientRequest — тело запроса для создания клиента.
type CreateClientRequest struct {
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Height    float64 `json:"height"`
	Weight    float64 `json:"weight"`
	Goal      string  `json:"goal"`
}

// UpdateClientRequest describes PATCH-style client updates.
type UpdateClientRequest struct {
	FirstName *string  `json:"first_name"`
	LastName  *string  `json:"last_name"`
	Height    *float64 `json:"height"`
	Weight    *float64 `json:"weight"`
	Goal      *string  `json:"goal"`
}

// ClientResponse — то, что возвращаем клиенту во всех клиентских эндпоинтах.
type ClientResponse struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`

	Height float64 `json:"height"`
	Weight float64 `json:"weight"`
	Goal   string  `json:"goal"`

	TotalSessions int `json:"total_sessions"`
	LeftSessions  int `json:"left_sessions"`
}
