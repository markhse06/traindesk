package app

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
	"traindesk/internal/DTO"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"traindesk/internal/config"
	"traindesk/internal/domain"
)

var cfg = config.Load()
var jwtSecret = []byte(cfg.JWTSecret)

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

// Вспомогательная функция для генерации случайной строки
func secureRandomString(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// handleRegister — регистрация пользователя.
func (a *App) handleRegister(c *gin.Context) {
	var req DTO.RegisterRequest

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

	u := domain.User{
		Email:         req.Email,
		PasswordHash:  string(hash),
		TrainerName:   req.TrainerName,
		EmailVerified: false,
	}

	if err := a.db.Create(&u).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "cannot create user (maybe email is already taken)",
			"details": err.Error(),
		})
		return
	}

	verification := domain.EmailVerification{
		UserID:    u.ID,
		Code:      code,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := a.mailer.SendEmailVerificationCode(code, req.TrainerName, req.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to send email verification code",
			"details": err.Error(),
		})
		a.db.Delete(&u)
		return
	}

	a.db.Create(&verification)

	resp := DTO.RegisterResponse{
		ID:          u.ID.String(),
		Email:       u.Email,
		TrainerName: u.TrainerName,
	}

	c.JSON(http.StatusCreated, resp)
}

// handleLogin — логин пользователя с проверкой пароля и статуса e‑mail.
func (a *App) handleLogin(c *gin.Context) {
	var req DTO.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	// Поиск пользователя
	var u domain.User
	if err := a.db.Where("email = ?", req.Email).First(&u).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	// Проверка верификации и пароля
	if !u.EmailVerified {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "email is not verified"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	// Генерация Access Token (короткий JWT)
	now := time.Now()
	accessClaims := jwt.MapClaims{
		"sub": u.ID.String(),
		"exp": now.Add(15 * time.Minute).Unix(),
		"iat": now.Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}

	// Генерация Refresh Token (длинная случайная строка)
	refreshTokenString := secureRandomString(32)

	refresh := domain.RefreshToken{
		UserID:    u.ID,
		Token:     refreshTokenString,
		ExpiresAt: now.Add(7 * 24 * time.Hour), // Живет 7 дней
	}

	// Сохраняем Refresh Token в БД
	if err := a.db.Create(&refresh).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save refresh token"})
		return
	}

	// Ответ
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessTokenString,
		"refresh_token": refreshTokenString,
		"user": gin.H{
			"id":           u.ID.String(),
			"email":        u.Email,
			"trainer_name": u.TrainerName,
		},
	})
}

// handleVerifyEmail — подтверждение e‑mail по коду.
func (a *App) handleVerifyEmail(c *gin.Context) {
	var req DTO.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	var u domain.User
	if err := a.db.Where("email = ?", req.Email).First(&u).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_not_found"})
		return
	}

	var v domain.EmailVerification
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

// handleChangePassword — метод для изменения пароля
func (a *App) handleChangePassword(c *gin.Context) {
	userIDStr := c.MustGet("user_id").(string)
	userID, _ := uuid.Parse(userIDStr)

	var req DTO.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	var u domain.User
	if err := a.db.First(&u, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user_not_found"})
		return
	}

	// Проверяем старый пароль
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_old_password"})
		return
	}

	// Хешируем новый
	newHash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 10)
	a.db.Model(&u).Update("password_hash", string(newHash))

	c.JSON(http.StatusOK, gin.H{"message": "password_changed"})
}

func (a *App) handleForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}

	var u domain.User
	if err := a.db.Where("email = ?", req.Email).First(&u).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "email doesn't exist"})
		return
	}

	code, _ := generateVerificationCode()

	// Сохраняем код в таблицу верификации
	reset := domain.EmailVerification{
		UserID:    u.ID,
		Code:      code,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	a.db.Create(&reset)

	var name string
	a.db.Raw("SELECT trainer_name FROM users WHERE id = ?", reset.UserID).Scan(&name)

	if err := a.mailer.SendEmailVerificationCode(code, req.Email, name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "reset code sent"})
}

func (a *App) handleResetPasswordConfirm(c *gin.Context) {
	var req DTO.ResetPasswordConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	var u domain.User
	if err := a.db.Where("email = ?", req.Email).First(&u).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_not_found"})
		return
	}

	var v domain.EmailVerification
	if err := a.db.Where("user_id = ? AND code = ?", u.ID, req.Code).First(&v).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_code"})
		return
	}

	if time.Now().After(v.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code_expired"})
		return
	}

	newHash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 10)
	a.db.Model(&u).Update("password_hash", string(newHash))

	a.db.Delete(&v)

	c.JSON(http.StatusOK, gin.H{"message": "password_reset_success"})
}

func (a *App) handleGetProfile(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	var u domain.User
	if err := a.db.First(&u, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user_not_found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           u.ID,
		"email":        u.Email,
		"trainer_name": u.TrainerName,
	})
}

func (a *App) handleRefresh(c *gin.Context) {
	var req DTO.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token is required"})
		return
	}

	// Ищем токен в базе данных
	var storedToken domain.RefreshToken
	if err := a.db.Where("token = ?", req.RefreshToken).First(&storedToken).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	// Проверяем, не протух ли он
	if time.Now().After(storedToken.ExpiresAt) {
		a.db.Delete(&storedToken) // Удаляем старый, чтобы не засорять базу
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token expired"})
		return
	}

	// Получаем пользователя
	var u domain.User
	if err := a.db.First(&u, "id = ?", storedToken.UserID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	// Генерируем новую пару токенов
	now := time.Now()

	// Новый Access Token
	accessClaims := jwt.MapClaims{
		"sub": u.ID.String(),
		"exp": now.Add(15 * time.Minute).Unix(),
		"iat": now.Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}

	// Новый Refresh Token
	newRefreshTokenString := secureRandomString(32)

	// Обновляем запись в базе или удаляем старую и создаем новую
	err = a.db.Transaction(func(tx *gorm.DB) error {
		// Удаляем использованный токен
		if err := tx.Delete(&storedToken).Error; err != nil {
			return err
		}

		// Создаем новый
		newRefresh := domain.RefreshToken{
			UserID:    u.ID,
			Token:     newRefreshTokenString,
			ExpiresAt: now.Add(7 * 24 * time.Hour),
		}
		return tx.Create(&newRefresh).Error
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update refresh token"})
		return
	}

	// Отдаем новые данные
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessTokenString,
		"refresh_token": newRefreshTokenString,
	})
}
