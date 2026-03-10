package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/fenco/trademate/services/api/internal/ads"
	"github.com/fenco/trademate/services/api/internal/auth"
	"github.com/fenco/trademate/services/api/internal/models"
	"github.com/fenco/trademate/services/api/internal/store"
	"github.com/fenco/trademate/services/api/internal/worker"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Handlers struct {
	repo         *store.Repository
	tokenService *auth.Service
	adsClient    *ads.Client
	worker       *worker.Service
	hub          *WebSocketHub
	upgrader     websocket.Upgrader
}

func NewHandlers(repo *store.Repository, tokenService *auth.Service, hub *WebSocketHub, adsClient *ads.Client, workerService *worker.Service) *Handlers {
	return &Handlers{
		repo:         repo,
		tokenService: tokenService,
		adsClient:    adsClient,
		worker:       workerService,
		hub:          hub,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(_ *http.Request) bool {
				return true
			},
		},
	}
}

func (h *Handlers) Health(c *gin.Context) {
	respond(c, http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handlers) Login(c *gin.Context) {
	var input models.LoginRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		respondErrorCode(c, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	user, storeID, roleCode, err := h.repo.Login(strings.TrimSpace(input.Account), input.Password)
	if err != nil {
		if errors.Is(err, store.ErrUnauthorized) {
			respondErrorCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid credentials")
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	token, err := h.tokenService.Sign(user.ID, storeID, roleCode)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to issue token")
		return
	}

	respond(c, http.StatusOK, models.LoginResponse{
		Token: token,
		User:  user,
	})
}

func (h *Handlers) Me(c *gin.Context) {
	userID := contextValue(c, ctxUserIDKey)
	activeStoreID := contextValue(c, ctxActiveStoreKey)

	data, err := h.repo.GetMe(userID, activeStoreID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondErrorCode(c, http.StatusNotFound, "NOT_FOUND", "user not found")
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	respond(c, http.StatusOK, data)
}

func (h *Handlers) ListStores(c *gin.Context) {
	userID := contextValue(c, ctxUserIDKey)

	stores, err := h.repo.ListStoresByUser(userID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	respond(c, http.StatusOK, gin.H{
		"list":  stores,
		"total": len(stores),
	})
}

func (h *Handlers) GetGoal(c *gin.Context) {
	storeID := contextValue(c, ctxActiveStoreKey)

	goal, err := h.repo.GetCurrentGoal(storeID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondErrorCode(c, http.StatusNotFound, "NOT_FOUND", "goal not found")
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	respond(c, http.StatusOK, goal)
}

func (h *Handlers) ListGoals(c *gin.Context) {
	storeID := contextValue(c, ctxActiveStoreKey)

	goals, err := h.repo.ListGoals(storeID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	respond(c, http.StatusOK, gin.H{
		"list":  goals,
		"total": len(goals),
	})
}

func (h *Handlers) CreateGoal(c *gin.Context) {
	storeID := contextValue(c, ctxActiveStoreKey)
	userID := contextValue(c, ctxUserIDKey)

	var input models.UpdateGoalRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		respondErrorCode(c, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}
	if err := validateGoalInput(input); err != nil {
		respondErrorCode(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}

	goal, err := h.repo.CreateGoal(storeID, userID, input)
	if err != nil {
		h.handleStoreError(c, err)
		return
	}

	respond(c, http.StatusCreated, goal)
}

func (h *Handlers) UpdateGoal(c *gin.Context) {
	storeID := contextValue(c, ctxActiveStoreKey)
	userID := contextValue(c, ctxUserIDKey)
	goalID := c.Param("goal_id")

	var input models.UpdateGoalRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		respondErrorCode(c, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}
	if err := validateGoalInput(input); err != nil {
		respondErrorCode(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}

	goal, err := h.repo.UpdateGoalByID(storeID, userID, goalID, input)
	if err != nil {
		h.handleStoreError(c, err)
		return
	}

	respond(c, http.StatusOK, goal)
}

func (h *Handlers) DeleteGoal(c *gin.Context) {
	storeID := contextValue(c, ctxActiveStoreKey)
	userID := contextValue(c, ctxUserIDKey)
	goalID := c.Param("goal_id")

	if err := h.repo.DeleteGoalByID(storeID, userID, goalID); err != nil {
		h.handleStoreError(c, err)
		return
	}

	respond(c, http.StatusOK, gin.H{
		"goal_id": goalID,
		"status":  "paused",
	})
}

func (h *Handlers) UpsertGoal(c *gin.Context) {
	storeID := contextValue(c, ctxActiveStoreKey)
	userID := contextValue(c, ctxUserIDKey)

	var input models.UpdateGoalRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		respondErrorCode(c, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	if err := validateGoalInput(input); err != nil {
		respondErrorCode(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}

	goal, err := h.repo.UpsertCurrentGoal(storeID, userID, input)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	respond(c, http.StatusOK, goal)
}

func (h *Handlers) ListSuggestions(c *gin.Context) {
	storeID := contextValue(c, ctxActiveStoreKey)
	payload, err := h.repo.ListSuggestions(store.SuggestionFilter{
		StoreID:   storeID,
		SiteCode:  c.Query("site_code"),
		Status:    c.Query("status"),
		RiskLevel: c.Query("risk_level"),
		Page:      parseIntOrDefault(c.Query("page"), 1),
		PageSize:  parseIntOrDefault(c.Query("page_size"), 20),
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	respond(c, http.StatusOK, payload)
}

func (h *Handlers) GetAdsDataPreview(c *gin.Context) {
	storeID := contextValue(c, ctxActiveStoreKey)

	data, err := h.adsClient.FetchPreviewData(c.Request.Context(), storeID)
	if err != nil {
		respondErrorCode(c, http.StatusBadGateway, "ADS_UPSTREAM_ERROR", err.Error())
		return
	}

	respond(c, http.StatusOK, data)
}

func (h *Handlers) GetSuggestionDetail(c *gin.Context) {
	storeID := contextValue(c, ctxActiveStoreKey)
	suggestionID := c.Param("suggestion_id")

	suggestion, err := h.repo.GetSuggestionByID(storeID, suggestionID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondErrorCode(c, http.StatusNotFound, "NOT_FOUND", "suggestion not found")
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	goal, err := h.repo.GetCurrentGoal(storeID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	approval, err := h.repo.GetLatestApprovalBySuggestion(suggestionID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	respond(c, http.StatusOK, gin.H{
		"suggestion":              suggestion,
		"goal":                    goal,
		"latest_context_snapshot": nil,
		"approval_preview":        approval,
	})
}

func (h *Handlers) ApproveSuggestion(c *gin.Context) {
	storeID := contextValue(c, ctxActiveStoreKey)
	userID := contextValue(c, ctxUserIDKey)
	suggestionID := c.Param("suggestion_id")

	var input models.ApproveSuggestionRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		respondErrorCode(c, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	result, err := h.repo.ApproveSuggestion(storeID, userID, suggestionID, input)
	if err != nil {
		h.handleStoreError(c, err)
		return
	}

	targetType := "task"
	targetID := result.TaskID
	_ = h.repo.CreateNotificationForStore(storeID, "task_update", "medium", "Suggestion approved", "A suggestion has been approved and turned into a task.", &targetType, &targetID)

	h.publishTaskStatusChanged(storeID, result.TaskID, result.TaskStatus)
	respond(c, http.StatusOK, result)
}

func (h *Handlers) RejectSuggestion(c *gin.Context) {
	storeID := contextValue(c, ctxActiveStoreKey)
	userID := contextValue(c, ctxUserIDKey)
	suggestionID := c.Param("suggestion_id")

	var input models.RejectSuggestionRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		respondErrorCode(c, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	if err := h.repo.RejectSuggestion(storeID, userID, suggestionID, input); err != nil {
		h.handleStoreError(c, err)
		return
	}

	targetType := "suggestion"
	targetID := suggestionID
	_ = h.repo.CreateNotificationForStore(storeID, "suggestion_update", "low", "Suggestion rejected", "A suggestion has been rejected.", &targetType, &targetID)
	h.publishNotificationEvent(storeID)
	respond(c, http.StatusOK, gin.H{"suggestion_id": suggestionID, "status": "rejected"})
}

func (h *Handlers) BatchApproveSuggestions(c *gin.Context) {
	storeID := contextValue(c, ctxActiveStoreKey)
	userID := contextValue(c, ctxUserIDKey)

	var input models.BatchApproveRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		respondErrorCode(c, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}
	if len(input.SuggestionIDs) == 0 {
		respondErrorCode(c, http.StatusBadRequest, "INVALID_PARAMS", "suggestion_ids is required")
		return
	}

	results, err := h.repo.BatchApproveSuggestions(storeID, userID, input)
	if err != nil {
		h.handleStoreError(c, err)
		return
	}

	targetType := "task"
	_ = h.repo.CreateNotificationForStore(storeID, "task_update", "medium", "Batch approval completed", "Suggestions were approved in batch.", &targetType, nil)
	h.publishNotificationEvent(storeID)

	for _, item := range results {
		h.publishTaskStatusChanged(storeID, item.TaskID, item.TaskStatus)
	}

	respond(c, http.StatusOK, gin.H{"results": results, "total": len(results)})
}

func (h *Handlers) ListTasks(c *gin.Context) {
	storeID := contextValue(c, ctxActiveStoreKey)
	tasks, total, err := h.repo.ListTasks(store.TaskFilter{
		StoreID:   storeID,
		Status:    c.Query("status"),
		RiskLevel: c.Query("risk_level"),
		Page:      parseIntOrDefault(c.Query("page"), 1),
		PageSize:  parseIntOrDefault(c.Query("page_size"), 20),
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	respond(c, http.StatusOK, gin.H{"list": tasks, "total": total})
}

func (h *Handlers) GetTask(c *gin.Context) {
	storeID := contextValue(c, ctxActiveStoreKey)
	taskID := c.Param("task_id")

	task, err := h.repo.GetTask(storeID, taskID)
	if err != nil {
		h.handleStoreError(c, err)
		return
	}
	events, err := h.repo.ListTaskEvents(taskID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	auditLogs, err := h.repo.ListAuditLogs(storeID, 50)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	reviewStatus := "pending"
	snapshot, err := h.repo.GetReviewSnapshot(storeID, taskID)
	if err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
	} else {
		reviewStatus = snapshot.Status
	}

	respond(c, http.StatusOK, models.TaskDetailResponse{
		Task:         task,
		TaskEvents:   events,
		AuditLogs:    auditLogs,
		ReviewStatus: reviewStatus,
	})
}

func (h *Handlers) GetTaskReview(c *gin.Context) {
	storeID := contextValue(c, ctxActiveStoreKey)
	taskID := c.Param("task_id")

	task, err := h.repo.GetTask(storeID, taskID)
	if err != nil {
		h.handleStoreError(c, err)
		return
	}

	snapshot, err := h.repo.GetReviewSnapshot(storeID, taskID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respond(c, http.StatusOK, models.ReviewSnapshot{
				AgentType:     task.AgentType,
				TaskID:        taskID,
				StoreID:       storeID,
				Status:        "pending",
				BeforeMetrics: map[string]any{},
			})
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	respond(c, http.StatusOK, snapshot)
}

func (h *Handlers) CancelTask(c *gin.Context) {
	storeID := contextValue(c, ctxActiveStoreKey)
	userID := contextValue(c, ctxUserIDKey)
	taskID := c.Param("task_id")

	task, err := h.repo.UpdateTaskStatus(storeID, userID, taskID, "cancelled", "cancelled by user")
	if err != nil {
		h.handleStoreError(c, err)
		return
	}

	publishTaskEvent(h, storeID, taskID, task.Status)
	respond(c, http.StatusOK, task)
}

func (h *Handlers) RetryTask(c *gin.Context) {
	storeID := contextValue(c, ctxActiveStoreKey)
	userID := contextValue(c, ctxUserIDKey)
	taskID := c.Param("task_id")

	task, err := h.repo.UpdateTaskStatus(storeID, userID, taskID, "queued", "retry queued")
	if err != nil {
		h.handleStoreError(c, err)
		return
	}

	publishTaskEvent(h, storeID, taskID, task.Status)
	respond(c, http.StatusOK, task)
}

func (h *Handlers) RunTasksOnce(c *gin.Context) {
	if h.worker == nil {
		respondErrorCode(c, http.StatusServiceUnavailable, "WORKER_DISABLED", "worker service unavailable")
		return
	}

	storeID := contextValue(c, ctxActiveStoreKey)
	userID := contextValue(c, ctxUserIDKey)

	var input struct {
		Limit int `json:"limit"`
	}
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrorCode(c, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
			return
		}
	}

	result, err := h.worker.RunOnce(c.Request.Context(), worker.RunOnceInput{
		StoreID: storeID,
		Limit:   input.Limit,
		ActorID: userID,
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	for _, item := range result.Results {
		if item.Status == "succeeded" || item.Status == "failed" {
			publishTaskEvent(h, item.StoreID, item.TaskID, item.Status)
		}
	}

	respond(c, http.StatusOK, result)
}

func (h *Handlers) ListNotifications(c *gin.Context) {
	userID := contextValue(c, ctxUserIDKey)
	limit := parseIntOrDefault(c.Query("limit"), 50)

	list, err := h.repo.ListNotifications(userID, limit)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	respond(c, http.StatusOK, gin.H{"list": list, "total": len(list)})
}

func (h *Handlers) MarkNotificationRead(c *gin.Context) {
	userID := contextValue(c, ctxUserIDKey)
	notificationID := c.Param("notification_id")

	if err := h.repo.MarkNotificationRead(userID, notificationID); err != nil {
		h.handleStoreError(c, err)
		return
	}

	respond(c, http.StatusOK, gin.H{"notification_id": notificationID, "is_read": true})
}

func (h *Handlers) ListAuditLogs(c *gin.Context) {
	storeID := contextValue(c, ctxActiveStoreKey)
	limit := parseIntOrDefault(c.Query("limit"), 100)

	list, err := h.repo.ListAuditLogs(storeID, limit)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	respond(c, http.StatusOK, gin.H{"list": list, "total": len(list)})
}

func (h *Handlers) WebSocket(c *gin.Context) {
	userID := contextValue(c, ctxUserIDKey)
	if userID == "" {
		respondErrorCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token")
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		respondErrorCode(c, http.StatusBadRequest, "WS_UPGRADE_FAILED", "failed to upgrade websocket")
		return
	}

	h.hub.Register(userID, conn)
	defer h.hub.Unregister(userID, conn)

	_ = conn.WriteJSON(EventMessage{Event: "connected", Payload: gin.H{"user_id": userID}})

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *Handlers) handleStoreError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		respondErrorCode(c, http.StatusNotFound, "NOT_FOUND", "resource not found")
	case errors.Is(err, store.ErrApprovalRequired):
		respondErrorCode(c, http.StatusBadRequest, "TASK_APPROVAL_REQUIRED", err.Error())
	case errors.Is(err, store.ErrInvalidTransition):
		respondErrorCode(c, http.StatusBadRequest, "INVALID_STATUS_TRANSITION", err.Error())
	case errors.Is(err, store.ErrUnauthorized):
		respondErrorCode(c, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
	default:
		respondError(c, http.StatusInternalServerError, err.Error())
	}
}

func (h *Handlers) publishTaskStatusChanged(storeID, taskID, status string) {
	userIDs, err := h.repo.ListUserIDsByStore(storeID)
	if err != nil {
		return
	}

	h.hub.PublishToUsers(userIDs, "task_status_changed", gin.H{
		"task_id": taskID,
		"status":  status,
	})
	h.publishNotificationEvent(storeID)
}

func (h *Handlers) publishNotificationEvent(storeID string) {
	userIDs, err := h.repo.ListUserIDsByStore(storeID)
	if err != nil {
		return
	}
	h.hub.PublishToUsers(userIDs, "notification_created", gin.H{"store_id": storeID})
}

func publishTaskEvent(h *Handlers, storeID, taskID, status string) {
	h.publishTaskStatusChanged(storeID, taskID, status)
}

func validateGoalInput(input models.UpdateGoalRequest) error {
	if strings.TrimSpace(input.GoalName) == "" || strings.TrimSpace(input.RiskProfile) == "" {
		return errors.New("goal_name and risk_profile are required")
	}
	return nil
}

func parseIntOrDefault(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
