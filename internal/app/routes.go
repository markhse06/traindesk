package app

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (a *App) registerRoutes() {
	a.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := a.router.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", a.handleRegister)
			auth.POST("/login", a.handleLogin)
			auth.POST("/verify-email", a.handleVerifyEmail)
		}

		workouts := api.Group("/workouts", a.AuthMiddleware())
		{
			workouts.GET("", a.handleGetWorkouts)
			workouts.POST("", a.handleCreateWorkout)
			workouts.GET("/:id", a.handleGetWorkoutByID)
			workouts.PUT("/:id", a.handleUpdateWorkout)
			workouts.DELETE("/:id", a.handleDeleteWorkout)
		}

		clients := api.Group("/clients", a.AuthMiddleware())
		{
			clients.GET("", a.handleGetClients)
			clients.POST("", a.handleCreateClient)
		}
	}
}
