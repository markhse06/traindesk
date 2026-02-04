package app

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"traindesk/internal/config"
	"traindesk/internal/user"
)

var cfg = config.Load()
var jwtSecret = []byte(cfg.JWTSecret)

const jwtTTL = 7 * 24 * time.Hour

// generateVerifyCode генерирует короткий код подтверждения почты.
func generateVerificationCode() (string, error) {
	// 0..999999
	var b [3]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	// используем первые 3 байта как число
	n := int(b[0])<<16 | int(b[1])<<8 | int(b[2])
	code := n % 1000000
	return fmt.Sprintf("%06d", code), nil // всегда 6 цифр, с лидирующими нулями
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
	code, err := generateVerificationCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate verify code"})
		return
	}

	u := user.User{
		Email:         req.Email,
		PasswordHash:  string(hash),
		TrainerName:   req.TrainerName,
		EmailVerified: false,
	}

	verification := user.EmailVerification{
		UserID:    u.ID,
		Code:      code,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	a.db.Create(&verification)

	if err := a.db.Create(&u).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "cannot create user (maybe email is already taken)",
			"details": err.Error(),
		})
		return
	}

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

	if !u.EmailVerified {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "email is not verified"})
		return
	}

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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	var u user.User
	if err := a.db.Where("email = ?", req.Email).First(&u).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_not_found"})
		return
	}

	var v user.EmailVerification
	if err := a.db.Where("user_id = ? AND code = ?", u.ID, req.Code).First(&v).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_code"})
		return
	}

	if time.Now().After(v.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code_expired"})
		return
	}

	if err := a.db.Model(&u).Update("email_verified", true).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed_to_verify"})
		return
	}

	a.db.Delete(&v)

	c.JSON(http.StatusOK, gin.H{"message": "email_verified"})
}
