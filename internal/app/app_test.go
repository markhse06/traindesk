package app

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"traindesk/internal/user"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Загружаем именно тестовый конфиг.
	// Т.к. тест лежит в internal/app, нужно подняться на 2 уровня выше к корню.
	err := godotenv.Load("../../.env.test")
	if err != nil {
		log.Fatal("Error loading .env.test file")
	}
}

func setupTestApp(t *testing.T) *App {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("Failed to init app: %v", err)
	}

	// Очищаем таблицы перед каждым тестом, чтобы не было конфликтов по Email/ID
	app.db.Exec("DELETE FROM email_verifications")
	app.db.Exec("DELETE FROM workout_clients")
	app.db.Exec("DELETE FROM workouts")
	app.db.Exec("DELETE FROM clients")
	app.db.Exec("DELETE FROM users")

	return app
}

func TestRegisterSuccess(t *testing.T) {
	app := setupTestApp(t)

	// 1. Подготавливаем данные
	regReq := user.RegisterRequest{
		Email:       "test@test.com",
		Password:    "password123",
		TrainerName: "Coach Ivan",
	}
	body, _ := json.Marshal(regReq)

	// 2. Создаем фейковый HTTP-запрос
	req, _ := http.NewRequest("POST", "/api/v1/user/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// 3. Создаем "записыватель" ответа
	w := httptest.NewRecorder()

	// 4. Запускаем обработку запроса
	app.router.ServeHTTP(w, req)

	// 5. Проверяем результат (Assertions)
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp user.RegisterResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "test@test.com", resp.Email)
	assert.NotEmpty(t, resp.ID)
}

func TestRegisterDuplicateEmail(t *testing.T) {
	app := setupTestApp(t)

	regReq := user.RegisterRequest{
		Email:       "double@test.com",
		Password:    "password123",
		TrainerName: "Coach",
	}
	body, _ := json.Marshal(regReq)

	// Регистрируем первый раз
	req1, _ := http.NewRequest("POST", "/api/v1/user/register", bytes.NewBuffer(body))
	w1 := httptest.NewRecorder()
	app.router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusCreated, w1.Code)

	// Пытаемся зарегистрировать тот же email второй раз
	req2, _ := http.NewRequest("POST", "/api/v1/user/register", bytes.NewBuffer(body))
	w2 := httptest.NewRecorder()
	app.router.ServeHTTP(w2, req2)

	// Проверяем, что вернулась ошибка
	assert.Equal(t, http.StatusBadRequest, w2.Code)
}
