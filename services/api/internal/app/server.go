package app

import (
	"log"
	"os"

	"github.com/fenco/trademate/services/api/internal/ads"
	"github.com/fenco/trademate/services/api/internal/auth"
	"github.com/fenco/trademate/services/api/internal/config"
	"github.com/fenco/trademate/services/api/internal/executor"
	httpapi "github.com/fenco/trademate/services/api/internal/http"
	"github.com/fenco/trademate/services/api/internal/openclaw"
	"github.com/fenco/trademate/services/api/internal/store"
	"github.com/fenco/trademate/services/api/internal/worker"
	"github.com/gin-gonic/gin"
)

func NewServer(cfg config.Config) *gin.Engine {
	db, err := store.OpenDB(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("failed to connect mysql: %v", err)
	}

	wd, _ := os.Getwd()
	if err := store.ApplyMigrations(db, wd); err != nil {
		log.Fatalf("failed to apply migrations: %v", err)
	}
	if err := store.SeedDemoData(db); err != nil {
		log.Fatalf("failed to seed demo data: %v", err)
	}

	repo := store.NewRepository(db)
	tokenService := auth.NewService(cfg.JWTSecret, cfg.JWTExpiresHour)
	adsClient := ads.NewClient(cfg)
	executorRegistry := executor.NewDefaultRegistry()
	fallbackClient := openclaw.NewClient(cfg)
	workerService := worker.NewService(repo, executorRegistry, fallbackClient)
	hub := httpapi.NewWebSocketHub()
	handlers := httpapi.NewHandlers(repo, tokenService, hub, adsClient, workerService)

	router := gin.Default()
	router.Use(httpapi.CORSMiddlewareProxy())
	router.Use(httpapi.AuthMiddlewareProxy(tokenService))

	router.GET("/health", handlers.Health)

	api := router.Group("/api/v1")
	{
		api.POST("/auth/login", handlers.Login)
		api.GET("/me", handlers.Me)
		api.GET("/stores", handlers.ListStores)

		api.GET("/agent-goals/current", handlers.GetGoal)
		api.PATCH("/agent-goals/current", handlers.UpsertGoal)
		api.GET("/agent-goals", handlers.ListGoals)
		api.POST("/agent-goals", handlers.CreateGoal)
		api.PATCH("/agent-goals/:goal_id", handlers.UpdateGoal)
		api.DELETE("/agent-goals/:goal_id", handlers.DeleteGoal)

		api.GET("/agents/ad/suggestions", handlers.ListSuggestions)
		api.GET("/agents/ad/data-preview", handlers.GetAdsDataPreview)
		api.GET("/agents/ad/suggestions/:suggestion_id", handlers.GetSuggestionDetail)
		api.GET("/agents/ad/reviews", handlers.ListTaskReviews)
		api.GET("/agents/ad/reviews/:task_id", handlers.GetTaskReview)
		api.POST("/agents/ad/suggestions/:suggestion_id/approve", handlers.ApproveSuggestion)
		api.POST("/agents/ad/suggestions/:suggestion_id/reject", handlers.RejectSuggestion)
		api.POST("/agents/ad/suggestions/batch-approve", handlers.BatchApproveSuggestions)
		api.POST("/agents/ad/suggestions/batch-reject", handlers.BatchRejectSuggestions)

		api.GET("/tasks", handlers.ListTasks)
		api.POST("/tasks/run-once", handlers.RunTasksOnce)
		api.GET("/tasks/:task_id", handlers.GetTask)
		api.POST("/tasks/:task_id/cancel", handlers.CancelTask)
		api.POST("/tasks/:task_id/retry", handlers.RetryTask)

		api.GET("/notifications", handlers.ListNotifications)
		api.POST("/notifications/:notification_id/read", handlers.MarkNotificationRead)

		api.GET("/audit-logs", handlers.ListAuditLogs)
		api.GET("/ws", handlers.WebSocket)
	}

	return router
}
