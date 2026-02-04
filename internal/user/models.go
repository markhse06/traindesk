package user

// Если здесь уже есть User — оставь его в model.go, а тут только DTO.

// RegisterRequest описывает тело запроса для регистрации.
type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	TrainerName string `json:"trainer_name"`
}

// RegisterResponse описывает ответ при успешной регистрации.
type RegisterResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	TrainerName string `json:"trainer_name"`
}

// LoginRequest описывает тело запроса для логина.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse описывает ответ при логине.
type LoginResponse struct {
	Token       string `json:"token"`
	ID          string `json:"id"`
	Email       string `json:"email"`
	TrainerName string `json:"trainer_name"`
}

type VerifyEmailRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}
