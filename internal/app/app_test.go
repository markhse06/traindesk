package app

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
	"traindesk/internal/DTO"
	"traindesk/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestMain(m *testing.M) {
	if err := godotenv.Overload("../../.env.test"); err != nil {
		log.Fatal("Error loading .env.test file")
	}
	jwtSecret = []byte(os.Getenv("JWT_SECRET"))
	gin.SetMode(gin.TestMode)

	os.Exit(m.Run())
}

func setupTestApp(t *testing.T) *App {
	t.Helper()

	app, err := NewApp()
	require.NoError(t, err)

	require.NoError(t, app.db.Exec(`
		TRUNCATE TABLE
			email_verifications,
			refresh_tokens,
			workout_clients,
			workout_packages,
			workouts,
			clients,
			users
		RESTART IDENTITY CASCADE
	`).Error)

	return app
}

func stubMailer(app *App) {
	app.mailer.Client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"data":{"succeeded":1}}`)),
		}, nil
	})
}

func performJSONRequest(t *testing.T, app *App, method, path string, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var reqBody bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&reqBody).Encode(body))
	}

	req, err := http.NewRequest(method, path, &reqBody)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	app.router.ServeHTTP(w, req)
	return w
}

func createVerifiedTrainer(t *testing.T, app *App, email string) domain.User {
	t.Helper()

	passwordHash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	user := domain.User{
		ID:            uuid.New(),
		Email:         email,
		PasswordHash:  string(passwordHash),
		TrainerName:   "Integration Coach",
		EmailVerified: true,
	}
	require.NoError(t, app.db.Create(&user).Error)

	return user
}

func loginTrainer(t *testing.T, app *App, email string) string {
	t.Helper()

	resp := loginTrainerFull(t, app, email)
	return resp.AccessToken
}

func loginTrainerFull(t *testing.T, app *App, email string) DTO.LoginResponse {
	t.Helper()

	w := performJSONRequest(t, app, http.MethodPost, "/api/v1/user/login", "", DTO.LoginRequest{
		Email:    email,
		Password: "password123",
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp DTO.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.AccessToken)
	require.NotEmpty(t, resp.RefreshToken)

	return resp
}

func createClientViaAPI(t *testing.T, app *App, token string, firstName string) DTO.ClientResponse {
	t.Helper()

	w := performJSONRequest(t, app, http.MethodPost, "/api/v1/clients", token, DTO.CreateClientRequest{
		FirstName: firstName,
		LastName:  "Client",
		Height:    180,
		Weight:    80,
		Goal:      "General fitness",
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var client DTO.ClientResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &client))
	return client
}

func createWorkoutViaAPI(t *testing.T, app *App, token string, clientIDs []string, dateTime string) domain.Workout {
	t.Helper()

	w := performJSONRequest(t, app, http.MethodPost, "/api/v1/workouts", token, DTO.CreateWorkoutRequest{
		DateTime:    dateTime,
		DurationMin: 60,
		Type:        string(domain.WorkoutTypeStrength),
		ClientIDs:   clientIDs,
		Notes:       "integration workout",
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var workout domain.Workout
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &workout))
	require.NotEmpty(t, workout.ID)
	return workout
}

func TestAuthIntegration(t *testing.T) {
	app := setupTestApp(t)
	stubMailer(app)

	badRegister := performJSONRequest(t, app, http.MethodPost, "/api/v1/user/register", "", DTO.RegisterRequest{
		Email:       "bad@test.local",
		Password:    "123",
		TrainerName: "Bad Coach",
	})
	require.Equal(t, http.StatusBadRequest, badRegister.Code, badRegister.Body.String())

	registerResp := performJSONRequest(t, app, http.MethodPost, "/api/v1/user/register", "", DTO.RegisterRequest{
		Email:       "auth@test.local",
		Password:    "password123",
		TrainerName: "Auth Coach",
	})
	require.Equal(t, http.StatusCreated, registerResp.Code, registerResp.Body.String())

	duplicateResp := performJSONRequest(t, app, http.MethodPost, "/api/v1/user/register", "", DTO.RegisterRequest{
		Email:       "auth@test.local",
		Password:    "password123",
		TrainerName: "Auth Coach",
	})
	require.Equal(t, http.StatusBadRequest, duplicateResp.Code, duplicateResp.Body.String())

	unverifiedLogin := performJSONRequest(t, app, http.MethodPost, "/api/v1/user/login", "", DTO.LoginRequest{
		Email:    "auth@test.local",
		Password: "password123",
	})
	require.Equal(t, http.StatusUnauthorized, unverifiedLogin.Code, unverifiedLogin.Body.String())

	var user domain.User
	require.NoError(t, app.db.Where("email = ?", "auth@test.local").First(&user).Error)
	var verification domain.EmailVerification
	require.NoError(t, app.db.Where("user_id = ?", user.ID).First(&verification).Error)

	invalidVerify := performJSONRequest(t, app, http.MethodPost, "/api/v1/user/verify-email", "", DTO.VerifyEmailRequest{
		Email: "auth@test.local",
		Code:  "000000",
	})
	require.Equal(t, http.StatusBadRequest, invalidVerify.Code, invalidVerify.Body.String())

	verifyResp := performJSONRequest(t, app, http.MethodPost, "/api/v1/user/verify-email", "", DTO.VerifyEmailRequest{
		Email: "auth@test.local",
		Code:  verification.Code,
	})
	require.Equal(t, http.StatusOK, verifyResp.Code, verifyResp.Body.String())

	loginResp := loginTrainerFull(t, app, "auth@test.local")

	wrongPassword := performJSONRequest(t, app, http.MethodPost, "/api/v1/user/login", "", DTO.LoginRequest{
		Email:    "auth@test.local",
		Password: "wrong-password",
	})
	require.Equal(t, http.StatusUnauthorized, wrongPassword.Code, wrongPassword.Body.String())

	missingAuth := performJSONRequest(t, app, http.MethodGet, "/api/v1/user/profile", "", nil)
	require.Equal(t, http.StatusUnauthorized, missingAuth.Code, missingAuth.Body.String())

	profileResp := performJSONRequest(t, app, http.MethodGet, "/api/v1/user/profile", loginResp.AccessToken, nil)
	require.Equal(t, http.StatusOK, profileResp.Code, profileResp.Body.String())

	badChangePassword := performJSONRequest(t, app, http.MethodPost, "/api/v1/user/profile/change-password", loginResp.AccessToken, DTO.ChangePasswordRequest{
		OldPassword: "wrong-password",
		NewPassword: "newpassword123",
	})
	require.Equal(t, http.StatusUnauthorized, badChangePassword.Code, badChangePassword.Body.String())

	changePasswordResp := performJSONRequest(t, app, http.MethodPost, "/api/v1/user/profile/change-password", loginResp.AccessToken, DTO.ChangePasswordRequest{
		OldPassword: "password123",
		NewPassword: "newpassword123",
	})
	require.Equal(t, http.StatusOK, changePasswordResp.Code, changePasswordResp.Body.String())

	resetMissingUser := performJSONRequest(t, app, http.MethodPost, "/api/v1/user/forgot-password", "", DTO.ResetPasswordRequest{
		Email: "missing@test.local",
	})
	require.Equal(t, http.StatusNotFound, resetMissingUser.Code, resetMissingUser.Body.String())

	forgotResp := performJSONRequest(t, app, http.MethodPost, "/api/v1/user/forgot-password", "", DTO.ResetPasswordRequest{
		Email: "auth@test.local",
	})
	require.Equal(t, http.StatusOK, forgotResp.Code, forgotResp.Body.String())

	var reset domain.EmailVerification
	require.NoError(t, app.db.Where("user_id = ?", user.ID).Order("created_at desc").First(&reset).Error)

	badReset := performJSONRequest(t, app, http.MethodPost, "/api/v1/user/reset-password", "", DTO.ResetPasswordConfirmRequest{
		Email:       "auth@test.local",
		Code:        "111111",
		NewPassword: "resetpass123",
	})
	require.Equal(t, http.StatusBadRequest, badReset.Code, badReset.Body.String())

	resetResp := performJSONRequest(t, app, http.MethodPost, "/api/v1/user/reset-password", "", DTO.ResetPasswordConfirmRequest{
		Email:       "auth@test.local",
		Code:        reset.Code,
		NewPassword: "resetpass123",
	})
	require.Equal(t, http.StatusOK, resetResp.Code, resetResp.Body.String())

	refreshResp := performJSONRequest(t, app, http.MethodPost, "/api/v1/user/refresh", "", DTO.RefreshRequest{
		RefreshToken: loginResp.RefreshToken,
	})
	require.Equal(t, http.StatusOK, refreshResp.Code, refreshResp.Body.String())

	reuseRefreshResp := performJSONRequest(t, app, http.MethodPost, "/api/v1/user/refresh", "", DTO.RefreshRequest{
		RefreshToken: loginResp.RefreshToken,
	})
	require.Equal(t, http.StatusUnauthorized, reuseRefreshResp.Code, reuseRefreshResp.Body.String())
}

func TestClientsCRUDIntegration(t *testing.T) {
	app := setupTestApp(t)
	trainer := createVerifiedTrainer(t, app, "clients@test.local")
	otherTrainer := createVerifiedTrainer(t, app, "other@test.local")
	token := loginTrainer(t, app, trainer.Email)
	otherToken := loginTrainer(t, app, otherTrainer.Email)

	createResp := performJSONRequest(t, app, http.MethodPost, "/api/v1/clients", token, DTO.CreateClientRequest{
		FirstName: "Ivan",
		LastName:  "Petrov",
		Height:    182.5,
		Weight:    81.2,
		Goal:      "Strength",
	})
	require.Equal(t, http.StatusCreated, createResp.Code, createResp.Body.String())

	var created DTO.ClientResponse
	require.NoError(t, json.Unmarshal(createResp.Body.Bytes(), &created))
	require.NotEmpty(t, created.ID)
	require.Equal(t, trainer.ID.String(), created.UserID)
	require.Equal(t, "Ivan", created.FirstName)
	require.Equal(t, "Petrov", created.LastName)
	require.Equal(t, 182.5, created.Height)
	require.Equal(t, 81.2, created.Weight)
	require.Equal(t, "Strength", created.Goal)
	require.Equal(t, 0, created.TotalSessions)
	require.Equal(t, 0, created.LeftSessions)

	listResp := performJSONRequest(t, app, http.MethodGet, "/api/v1/clients", token, nil)
	require.Equal(t, http.StatusOK, listResp.Code, listResp.Body.String())

	var clients []DTO.ClientResponse
	require.NoError(t, json.Unmarshal(listResp.Body.Bytes(), &clients))
	require.Len(t, clients, 1)
	require.Equal(t, created.ID, clients[0].ID)

	otherTrainerResp := performJSONRequest(t, app, http.MethodGet, "/api/v1/clients/"+created.ID, otherToken, nil)
	require.Equal(t, http.StatusNotFound, otherTrainerResp.Code, otherTrainerResp.Body.String())

	newGoal := "Mobility"
	newWeight := 79.7
	updateResp := performJSONRequest(t, app, http.MethodPatch, "/api/v1/clients/"+created.ID, token, DTO.UpdateClientRequest{
		Weight: &newWeight,
		Goal:   &newGoal,
	})
	require.Equal(t, http.StatusOK, updateResp.Code, updateResp.Body.String())

	var updated DTO.ClientResponse
	require.NoError(t, json.Unmarshal(updateResp.Body.Bytes(), &updated))
	require.Equal(t, "Ivan", updated.FirstName)
	require.Equal(t, "Petrov", updated.LastName)
	require.Equal(t, 79.7, updated.Weight)
	require.Equal(t, "Mobility", updated.Goal)

	seedClientStats(t, app, trainer.ID, uuid.MustParse(created.ID))

	detailResp := performJSONRequest(t, app, http.MethodGet, "/api/v1/clients/"+created.ID, token, nil)
	require.Equal(t, http.StatusOK, detailResp.Code, detailResp.Body.String())

	var detail DTO.ClientResponse
	require.NoError(t, json.Unmarshal(detailResp.Body.Bytes(), &detail))
	require.Equal(t, 1, detail.TotalSessions)
	require.Equal(t, 7, detail.LeftSessions)

	deleteResp := performJSONRequest(t, app, http.MethodDelete, "/api/v1/clients/"+created.ID, token, nil)
	require.Equal(t, http.StatusNoContent, deleteResp.Code, deleteResp.Body.String())

	getDeletedResp := performJSONRequest(t, app, http.MethodGet, "/api/v1/clients/"+created.ID, token, nil)
	require.Equal(t, http.StatusNotFound, getDeletedResp.Code, getDeletedResp.Body.String())

	var workoutClientCount int64
	require.NoError(t, app.db.Model(&domain.WorkoutClient{}).Where("client_id = ?", created.ID).Count(&workoutClientCount).Error)
	require.Equal(t, int64(0), workoutClientCount)

	var packageCount int64
	require.NoError(t, app.db.Model(&domain.WorkoutPackage{}).Where("client_id = ?", created.ID).Count(&packageCount).Error)
	require.Equal(t, int64(0), packageCount)
}

func TestClientsNegativeIntegration(t *testing.T) {
	app := setupTestApp(t)
	trainer := createVerifiedTrainer(t, app, "clients-negative@test.local")
	token := loginTrainer(t, app, trainer.Email)

	invalidCreate := performJSONRequest(t, app, http.MethodPost, "/api/v1/clients", token, DTO.CreateClientRequest{
		FirstName: "",
		LastName:  "Client",
	})
	require.Equal(t, http.StatusBadRequest, invalidCreate.Code, invalidCreate.Body.String())

	notFound := performJSONRequest(t, app, http.MethodGet, "/api/v1/clients/"+uuid.NewString(), token, nil)
	require.Equal(t, http.StatusNotFound, notFound.Code, notFound.Body.String())

	invalidID := performJSONRequest(t, app, http.MethodGet, "/api/v1/clients/not-a-uuid", token, nil)
	require.Equal(t, http.StatusBadRequest, invalidID.Code, invalidID.Body.String())

	client := createClientViaAPI(t, app, token, "Negative")
	emptyFirstName := ""
	invalidPatch := performJSONRequest(t, app, http.MethodPatch, "/api/v1/clients/"+client.ID, token, DTO.UpdateClientRequest{
		FirstName: &emptyFirstName,
	})
	require.Equal(t, http.StatusBadRequest, invalidPatch.Code, invalidPatch.Body.String())

	invalidDeleteID := performJSONRequest(t, app, http.MethodDelete, "/api/v1/clients/not-a-uuid", token, nil)
	require.Equal(t, http.StatusBadRequest, invalidDeleteID.Code, invalidDeleteID.Body.String())
}

func TestWorkoutCRUDIntegration(t *testing.T) {
	app := setupTestApp(t)
	trainer := createVerifiedTrainer(t, app, "workouts@test.local")
	otherTrainer := createVerifiedTrainer(t, app, "workouts-other@test.local")
	token := loginTrainer(t, app, trainer.Email)
	otherToken := loginTrainer(t, app, otherTrainer.Email)

	client := createClientViaAPI(t, app, token, "Workout")
	secondClient := createClientViaAPI(t, app, token, "Second")
	dateTime := time.Now().UTC().Truncate(time.Second).Add(2 * time.Hour).Format(time.RFC3339)

	workout := createWorkoutViaAPI(t, app, token, []string{client.ID}, dateTime)

	listResp := performJSONRequest(t, app, http.MethodGet, "/api/v1/workouts", token, nil)
	require.Equal(t, http.StatusOK, listResp.Code, listResp.Body.String())
	var workouts []DTO.WorkoutResponse
	require.NoError(t, json.Unmarshal(listResp.Body.Bytes(), &workouts))
	require.Len(t, workouts, 1)
	require.Equal(t, workout.ID.String(), workouts[0].ID)
	require.Equal(t, []string{client.ID}, workouts[0].ClientIDs)

	filteredResp := performJSONRequest(t, app, http.MethodGet, "/api/v1/workouts?started_at="+dateTime+"&ended_at="+dateTime, token, nil)
	require.Equal(t, http.StatusOK, filteredResp.Code, filteredResp.Body.String())

	getResp := performJSONRequest(t, app, http.MethodGet, "/api/v1/workouts/"+workout.ID.String(), token, nil)
	require.Equal(t, http.StatusOK, getResp.Code, getResp.Body.String())

	otherGetResp := performJSONRequest(t, app, http.MethodGet, "/api/v1/workouts/"+workout.ID.String(), otherToken, nil)
	require.Equal(t, http.StatusNotFound, otherGetResp.Code, otherGetResp.Body.String())

	newDuration := 90
	newNotes := "updated notes"
	updateClients := []string{client.ID, secondClient.ID}
	updateResp := performJSONRequest(t, app, http.MethodPut, "/api/v1/workouts/"+workout.ID.String(), token, DTO.UpdateWorkoutRequest{
		DurationMin: &newDuration,
		Notes:       &newNotes,
		ClientIDs:   &updateClients,
	})
	require.Equal(t, http.StatusOK, updateResp.Code, updateResp.Body.String())

	updatedGetResp := performJSONRequest(t, app, http.MethodGet, "/api/v1/workouts/"+workout.ID.String(), token, nil)
	require.Equal(t, http.StatusOK, updatedGetResp.Code, updatedGetResp.Body.String())
	var updated DTO.WorkoutResponse
	require.NoError(t, json.Unmarshal(updatedGetResp.Body.Bytes(), &updated))
	require.ElementsMatch(t, updateClients, updated.ClientIDs)

	deleteResp := performJSONRequest(t, app, http.MethodDelete, "/api/v1/workouts/"+workout.ID.String(), token, nil)
	require.Equal(t, http.StatusNoContent, deleteResp.Code, deleteResp.Body.String())

	getDeletedResp := performJSONRequest(t, app, http.MethodGet, "/api/v1/workouts/"+workout.ID.String(), token, nil)
	require.Equal(t, http.StatusNotFound, getDeletedResp.Code, getDeletedResp.Body.String())

	var workoutClientCount int64
	require.NoError(t, app.db.Model(&domain.WorkoutClient{}).Where("workout_id = ?", workout.ID).Count(&workoutClientCount).Error)
	require.Equal(t, int64(0), workoutClientCount)
}

func TestWorkoutNegativeIntegration(t *testing.T) {
	app := setupTestApp(t)
	trainer := createVerifiedTrainer(t, app, "workouts-negative@test.local")
	token := loginTrainer(t, app, trainer.Email)
	client := createClientViaAPI(t, app, token, "WorkoutNegative")

	noClientsResp := performJSONRequest(t, app, http.MethodPost, "/api/v1/workouts", token, DTO.CreateWorkoutRequest{
		DateTime:    time.Now().UTC().Format(time.RFC3339),
		DurationMin: 60,
		Type:        string(domain.WorkoutTypeStrength),
	})
	require.Equal(t, http.StatusBadRequest, noClientsResp.Code, noClientsResp.Body.String())

	badDateResp := performJSONRequest(t, app, http.MethodPost, "/api/v1/workouts", token, DTO.CreateWorkoutRequest{
		DateTime:    "bad-date",
		DurationMin: 60,
		Type:        string(domain.WorkoutTypeStrength),
		ClientIDs:   []string{client.ID},
	})
	require.Equal(t, http.StatusBadRequest, badDateResp.Code, badDateResp.Body.String())

	invalidClientResp := performJSONRequest(t, app, http.MethodPost, "/api/v1/workouts", token, DTO.CreateWorkoutRequest{
		DateTime:    time.Now().UTC().Format(time.RFC3339),
		DurationMin: 60,
		Type:        string(domain.WorkoutTypeStrength),
		ClientIDs:   []string{uuid.NewString()},
	})
	require.Equal(t, http.StatusInternalServerError, invalidClientResp.Code, invalidClientResp.Body.String())

	invalidStartedAtResp := performJSONRequest(t, app, http.MethodGet, "/api/v1/workouts?started_at=bad-date", token, nil)
	require.Equal(t, http.StatusBadRequest, invalidStartedAtResp.Code, invalidStartedAtResp.Body.String())

	invalidEndedAtResp := performJSONRequest(t, app, http.MethodGet, "/api/v1/workouts?ended_at=bad-date", token, nil)
	require.Equal(t, http.StatusBadRequest, invalidEndedAtResp.Code, invalidEndedAtResp.Body.String())

	invalidGetIDResp := performJSONRequest(t, app, http.MethodGet, "/api/v1/workouts/not-a-uuid", token, nil)
	require.Equal(t, http.StatusBadRequest, invalidGetIDResp.Code, invalidGetIDResp.Body.String())

	workout := createWorkoutViaAPI(t, app, token, []string{client.ID}, time.Now().UTC().Add(time.Hour).Format(time.RFC3339))

	invalidDuration := 0
	invalidUpdateResp := performJSONRequest(t, app, http.MethodPut, "/api/v1/workouts/"+workout.ID.String(), token, DTO.UpdateWorkoutRequest{
		DurationMin: &invalidDuration,
	})
	require.Equal(t, http.StatusBadRequest, invalidUpdateResp.Code, invalidUpdateResp.Body.String())

	invalidType := "unknown"
	invalidTypeResp := performJSONRequest(t, app, http.MethodPut, "/api/v1/workouts/"+workout.ID.String(), token, DTO.UpdateWorkoutRequest{
		Type: &invalidType,
	})
	require.Equal(t, http.StatusBadRequest, invalidTypeResp.Code, invalidTypeResp.Body.String())

	invalidDeleteIDResp := performJSONRequest(t, app, http.MethodDelete, "/api/v1/workouts/not-a-uuid", token, nil)
	require.Equal(t, http.StatusBadRequest, invalidDeleteIDResp.Code, invalidDeleteIDResp.Body.String())
}

func TestWorkoutPackagesIntegration(t *testing.T) {
	app := setupTestApp(t)
	trainer := createVerifiedTrainer(t, app, "packages@test.local")
	token := loginTrainer(t, app, trainer.Email)
	client := createClientViaAPI(t, app, token, "Package")

	createPackageResp := performJSONRequest(t, app, http.MethodPost, "/api/v1/packages", token, DTO.CreatePackageRequest{
		ClientID:   client.ID,
		TotalCount: 2,
		Price:      1500,
	})
	require.Equal(t, http.StatusCreated, createPackageResp.Code, createPackageResp.Body.String())
	var pkg domain.WorkoutPackage
	require.NoError(t, json.Unmarshal(createPackageResp.Body.Bytes(), &pkg))
	require.Equal(t, 2, pkg.TotalCount)
	require.Equal(t, 0, pkg.UsedCount)
	require.True(t, pkg.IsActive)

	getPackagesResp := performJSONRequest(t, app, http.MethodGet, "/api/v1/packages/"+client.ID, token, nil)
	require.Equal(t, http.StatusOK, getPackagesResp.Code, getPackagesResp.Body.String())
	var packages []domain.WorkoutPackage
	require.NoError(t, json.Unmarshal(getPackagesResp.Body.Bytes(), &packages))
	require.Len(t, packages, 1)

	firstWorkoutResp := performJSONRequest(t, app, http.MethodPost, "/api/v1/workouts", token, DTO.CreateWorkoutRequest{
		DateTime:    time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
		DurationMin: 60,
		Type:        string(domain.WorkoutTypeStrength),
		ClientIDs:   []string{client.ID},
		PackageID:   stringPtr(pkg.ID.String()),
	})
	require.Equal(t, http.StatusCreated, firstWorkoutResp.Code, firstWorkoutResp.Body.String())

	secondWorkoutResp := performJSONRequest(t, app, http.MethodPost, "/api/v1/workouts", token, DTO.CreateWorkoutRequest{
		DateTime:    time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339),
		DurationMin: 60,
		Type:        string(domain.WorkoutTypeStrength),
		ClientIDs:   []string{client.ID},
		PackageID:   stringPtr(pkg.ID.String()),
	})
	require.Equal(t, http.StatusCreated, secondWorkoutResp.Code, secondWorkoutResp.Body.String())

	var updatedPackage domain.WorkoutPackage
	require.NoError(t, app.db.First(&updatedPackage, "id = ?", pkg.ID).Error)
	require.Equal(t, 2, updatedPackage.UsedCount)
	require.False(t, updatedPackage.IsActive)

	noSessionsLeftResp := performJSONRequest(t, app, http.MethodPost, "/api/v1/workouts", token, DTO.CreateWorkoutRequest{
		DateTime:    time.Now().UTC().Add(3 * time.Hour).Format(time.RFC3339),
		DurationMin: 60,
		Type:        string(domain.WorkoutTypeStrength),
		ClientIDs:   []string{client.ID},
		PackageID:   stringPtr(pkg.ID.String()),
	})
	require.Equal(t, http.StatusInternalServerError, noSessionsLeftResp.Code, noSessionsLeftResp.Body.String())

	activePackagesResp := performJSONRequest(t, app, http.MethodGet, "/api/v1/packages/"+client.ID, token, nil)
	require.Equal(t, http.StatusOK, activePackagesResp.Code, activePackagesResp.Body.String())
	var activePackages []domain.WorkoutPackage
	require.NoError(t, json.Unmarshal(activePackagesResp.Body.Bytes(), &activePackages))
	require.Empty(t, activePackages)
}

func TestWorkoutPackagesNegativeIntegration(t *testing.T) {
	app := setupTestApp(t)
	trainer := createVerifiedTrainer(t, app, "packages-negative@test.local")
	token := loginTrainer(t, app, trainer.Email)

	invalidJSONResp := performJSONRequest(t, app, http.MethodPost, "/api/v1/packages", token, map[string]any{
		"client_id": 123,
	})
	require.Equal(t, http.StatusBadRequest, invalidJSONResp.Code, invalidJSONResp.Body.String())

	invalidClientIDResp := performJSONRequest(t, app, http.MethodPost, "/api/v1/packages", token, DTO.CreatePackageRequest{
		ClientID:   "not-a-uuid",
		TotalCount: 5,
	})
	require.Equal(t, http.StatusBadRequest, invalidClientIDResp.Code, invalidClientIDResp.Body.String())
}

func stringPtr(s string) *string {
	return &s
}

func seedClientStats(t *testing.T, app *App, trainerID, clientID uuid.UUID) {
	t.Helper()

	activePackage := domain.WorkoutPackage{
		ID:         uuid.New(),
		TrainerID:  trainerID,
		ClientID:   clientID,
		TotalCount: 10,
		UsedCount:  3,
		IsActive:   true,
		Price:      1000,
	}
	inactivePackage := domain.WorkoutPackage{
		ID:         uuid.New(),
		TrainerID:  trainerID,
		ClientID:   clientID,
		TotalCount: 20,
		UsedCount:  0,
		IsActive:   false,
		Price:      2000,
	}
	require.NoError(t, app.db.Create(&activePackage).Error)
	require.NoError(t, app.db.Create(&inactivePackage).Error)
	require.NoError(t, app.db.Model(&inactivePackage).Update("is_active", false).Error)

	workout := domain.Workout{
		ID:          uuid.New(),
		UserID:      trainerID,
		PackageID:   activePackage.ID,
		DateTime:    time.Now().UTC(),
		DurationMin: 60,
		Type:        domain.WorkoutTypeStrength,
		Price:       0,
	}
	require.NoError(t, app.db.Create(&workout).Error)
	require.NoError(t, app.db.Create(&domain.WorkoutClient{
		WorkoutID: workout.ID,
		ClientID:  clientID,
	}).Error)
}
