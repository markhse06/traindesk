package user

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

// VerifyEmailRequest описывает тело запроса подтверждения почты.
type VerifyEmailRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// ChangePasswordRequest тело запроса смены пароля.
type ChangePasswordRequest struct {
	Email       string `json:"email"`
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// ChangePasswordResponse ответ на запрос изменения пароля.
type ChangePasswordResponse struct {
	Email     string `json:"email"`
	IsChanged bool   `json:"is_changed"`
}

// Сброс пароля
type ResetPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordByCodeRequest struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	NewPassword string `json:"new_password"`
}
