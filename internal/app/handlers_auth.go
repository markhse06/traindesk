package app

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"traindesk/internal/user"
)

// Секрет и время жизни токена (ТЗ: не более 7 дней).
// TODO поменять на ключ из .env
var jwtSecret = []byte("very-secret-key")

const jwtTTL = 7 * 24 * time.Hour

// generateVerifyCode генерирует короткий код подтверждения почты.
func generateVerifyCode() (string, error) {
	b := make([]byte, 4) // 8 hex-символов
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// handleRegister — регистрация пользователя.
func (a *App) handleRegister(c *gin.Context) {
	var req user.RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	// Валидация по ТЗ: email обязателен, пароль >= 6, имя тренера обязательно.
	if req.Email == "" || len(req.Password) < 6 || req.TrainerName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "email, password (>=6) and trainer_name are required",
		})
		return
	}

	// Хешируем пароль (bcrypt, cost >= 10).
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	// Генерируем код подтверждения почты.
	code, err := generateVerifyCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate verify code"})
		return
	}

	u := user.User{
		Email:         req.Email,
		PasswordHash:  string(hash),
		TrainerName:   req.TrainerName,
		EmailVerified: false,
		VerifyCode:    code,
	}

	if err := a.db.Create(&u).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "cannot create user (maybe email is already taken)",
			"details": err.Error(),
		})
		return
	}

	// В учебной версии возвращаем код в ответе, чтобы можно было подтвердить без e‑mail.
	resp := user.RegisterResponse{
		ID:          u.ID.String(),
		Email:       u.Email,
		TrainerName: u.TrainerName,
	}

	c.JSON(http.StatusCreated, resp)
}

// handleLogin — логин пользователя с проверкой пароля и статуса e‑mail.
func (a *App) handleLogin(c *gin.Context) {
	var req user.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	if req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and password are required"})
		return
	}

	var u user.User
	if err := a.db.Where("email = ?", req.Email).First(&u).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	// TODO: блок для верификации, закомментирован на время тестирования и разработки.
	// Не пускаем, если почта не подтверждена.
	// if !u.EmailVerified {
	//	c.JSON(http.StatusUnauthorized, gin.H{"error": "email is not verified"})
	//	return
	// }

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"sub": u.ID.String(),
		"exp": now.Add(jwtTTL).Unix(),
		"iat": now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	resp := user.LoginResponse{
		Token:       tokenString,
		ID:          u.ID.String(),
		Email:       u.Email,
		TrainerName: u.TrainerName,
	}

	c.JSON(http.StatusOK, resp)
}

// handleVerifyEmail — подтверждение e‑mail по коду.
func (a *App) handleVerifyEmail(c *gin.Context) {
	var req user.VerifyEmailRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	if req.Email == "" || req.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and code are required"})
		return
	}

	var u user.User
	if err := a.db.Where("email = ?", req.Email).First(&u).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user not found"})
		return
	}

	if u.EmailVerified {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email already verified"})
		return
	}

	if u.VerifyCode != req.Code {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid verification code"})
		return
	}

	u.EmailVerified = true
	u.VerifyCode = ""

	if err := a.db.Save(&u).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "email verified"})
}
