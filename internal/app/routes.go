package app

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func (a *App) registerRoutes() {
	a.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	a.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := a.router.Group("/api/v1")
	{
		auth := api.Group("/user")
		{
			auth.POST("/register", a.handleRegister)
			auth.POST("/login", a.handleLogin)
			auth.POST("/verify-email", a.handleVerifyEmail)
			auth.POST("/forgot-password", a.handleForgotPassword)
			auth.POST("/reset-password", a.handleResetPasswordConfirm)
			auth.POST("/refresh", a.handleRefresh)

			// Защищенные роуты профиля
			profile := auth.Group("/profile", a.AuthMiddleware())
			{
				profile.GET("", a.handleGetProfile)
				profile.POST("/change-password", a.handleChangePassword)
			}
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
