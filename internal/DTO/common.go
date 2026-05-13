package DTO

type ErrorResponse struct {
	Error string `json:"error"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type TokenPairResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type LoginResponse struct {
	AccessToken  string           `json:"access_token"`
	RefreshToken string           `json:"refresh_token"`
	User         RegisterResponse `json:"user"`
}

type ProfileResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	TrainerName string `json:"trainer_name"`
}
