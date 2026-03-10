package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fenco/trademate/services/api/internal/models"
)

type SuggestionFilter struct {
	StoreID   string
	SiteCode  string
	Status    string
	RiskLevel string
	Page      int
	PageSize  int
}

type TaskFilter struct {
	StoreID   string
	Status    string
	RiskLevel string
	Page      int
	PageSize  int
}

type QueuedTask struct {
	StoreID string
	Task    models.Task
}

func (r *Repository) Login(account, password string) (models.User, string, string, error) {
	row := r.db.QueryRow(`
SELECT id, email, phone, display_name, status, password_hash
FROM user_account
WHERE email = ? OR phone = ?
LIMIT 1`, account, account)

	var user models.User
	var email sql.NullString
	var phone sql.NullString
	var passwordHash string
	if err := row.Scan(&user.ID, &email, &phone, &user.DisplayName, &user.Status, &passwordHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, "", "", ErrUnauthorized
		}
		return models.User{}, "", "", err
	}

	if email.Valid {
		user.Email = &email.String
	}
	if phone.Valid {
		user.Phone = &phone.String
	}

	if passwordHash != password {
		return models.User{}, "", "", ErrUnauthorized
	}

	roleRow := r.db.QueryRow(`
SELECT store_id, role_code
FROM role_assignment
WHERE user_id = ?
ORDER BY created_at ASC
LIMIT 1`, user.ID)

	var storeID string
	var roleCode string
	if err := roleRow.Scan(&storeID, &roleCode); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, "", "", ErrUnauthorized
		}
		return models.User{}, "", "", err
	}

	return user, storeID, roleCode, nil
}

func (r *Repository) GetMe(userID, activeStoreID string) (models.MeResponse, error) {
	row := r.db.QueryRow(`
SELECT id, email, phone, display_name, status
FROM user_account
WHERE id = ?
LIMIT 1`, userID)

	var user models.User
	var email sql.NullString
	var phone sql.NullString
	if err := row.Scan(&user.ID, &email, &phone, &user.DisplayName, &user.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.MeResponse{}, ErrNotFound
		}
		return models.MeResponse{}, err
	}
	if email.Valid {
		user.Email = &email.String
	}
	if phone.Valid {
		user.Phone = &phone.String
	}

	roleRows, err := r.db.Query(`
SELECT id, user_id, store_id, role_code
FROM role_assignment
WHERE user_id = ?`, userID)
	if err != nil {
		return models.MeResponse{}, err
	}
	defer roleRows.Close()

	roles := make([]models.RoleAssignment, 0)
	storeIDs := make([]string, 0)
	storeSeen := map[string]struct{}{}
	for roleRows.Next() {
		var role models.RoleAssignment
		if err := roleRows.Scan(&role.ID, &role.UserID, &role.StoreID, &role.RoleCode); err != nil {
			return models.MeResponse{}, err
		}
		roles = append(roles, role)
		if _, exists := storeSeen[role.StoreID]; !exists {
			storeSeen[role.StoreID] = struct{}{}
			storeIDs = append(storeIDs, role.StoreID)
		}
	}

	stores := make([]models.Store, 0)
	for _, storeID := range storeIDs {
		storeRow := r.db.QueryRow(`
SELECT id, site_code, store_name, currency, timezone, status
FROM store
WHERE id = ?
LIMIT 1`, storeID)

		var store models.Store
		if err := storeRow.Scan(&store.ID, &store.SiteCode, &store.StoreName, &store.Currency, &store.Timezone, &store.Status); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return models.MeResponse{}, err
		}
		stores = append(stores, store)
	}

	if activeStoreID == "" && len(stores) > 0 {
		activeStoreID = stores[0].ID
	}

	return models.MeResponse{
		User:          user,
		Roles:         roles,
		Stores:        stores,
		ActiveStoreID: activeStoreID,
	}, nil
}

func (r *Repository) ListStoresByUser(userID string) ([]models.Store, error) {
	rows, err := r.db.Query(`
SELECT DISTINCT s.id, s.site_code, s.store_name, s.currency, s.timezone, s.status
FROM role_assignment r
INNER JOIN store s ON s.id = r.store_id
WHERE r.user_id = ?
ORDER BY s.store_name ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stores := make([]models.Store, 0)
	for rows.Next() {
		var store models.Store
		if err := rows.Scan(&store.ID, &store.SiteCode, &store.StoreName, &store.Currency, &store.Timezone, &store.Status); err != nil {
			return nil, err
		}
		stores = append(stores, store)
	}

	return stores, nil
}

func (r *Repository) GetCurrentGoal(storeID string) (models.AdGoal, error) {
	row := r.db.QueryRow(`
SELECT id, agent_type, store_id, site_code, goal_name, acos_target, daily_budget_cap,
       risk_profile, auto_approve_enabled, auto_approve_budget_delta_pct, auto_approve_bid_delta_pct,
       status, effective_from, updated_by
FROM agent_goal
WHERE store_id = ? AND status = 'active'
ORDER BY updated_at DESC
LIMIT 1`, storeID)

	goal, err := scanGoalRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.AdGoal{}, ErrNotFound
		}
		return models.AdGoal{}, err
	}

	return goal, nil
}

func (r *Repository) UpsertCurrentGoal(storeID, userID string, input models.UpdateGoalRequest) (models.AdGoal, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return models.AdGoal{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	goalID := ""
	row := tx.QueryRow(`
SELECT id
FROM agent_goal
WHERE store_id = ? AND status = 'active'
ORDER BY updated_at DESC
LIMIT 1
FOR UPDATE`, storeID)
	_ = row.Scan(&goalID)

	if goalID == "" {
		goalID = newID("goal")
		_, err = tx.Exec(`
INSERT INTO agent_goal (
  id, agent_type, store_id, site_code, goal_name, acos_target, daily_budget_cap,
  risk_profile, auto_approve_enabled, auto_approve_budget_delta_pct, auto_approve_bid_delta_pct,
  status, effective_from, updated_by, created_at, updated_at
)
VALUES (?, 'ad_agent', ?, 'US', ?, ?, ?, ?, ?, ?, ?, 'active', UTC_TIMESTAMP(), ?, UTC_TIMESTAMP(), UTC_TIMESTAMP())`,
			goalID, storeID, input.GoalName, input.ACOSTarget, input.DailyBudgetCap,
			input.RiskProfile, input.AutoApproveEnabled, input.AutoApproveBudgetDeltaPct, input.AutoApproveBidDeltaPct,
			userID)
		if err != nil {
			return models.AdGoal{}, err
		}
	} else {
		_, err = tx.Exec(`
UPDATE agent_goal
SET goal_name = ?,
    acos_target = ?,
    daily_budget_cap = ?,
    risk_profile = ?,
    auto_approve_enabled = ?,
    auto_approve_budget_delta_pct = ?,
    auto_approve_bid_delta_pct = ?,
    effective_from = UTC_TIMESTAMP(),
    updated_by = ?,
    updated_at = UTC_TIMESTAMP()
WHERE id = ?`,
			input.GoalName, input.ACOSTarget, input.DailyBudgetCap, input.RiskProfile,
			input.AutoApproveEnabled, input.AutoApproveBudgetDeltaPct, input.AutoApproveBidDeltaPct,
			userID, goalID)
		if err != nil {
			return models.AdGoal{}, err
		}
	}

	goalRow := tx.QueryRow(`
SELECT id, agent_type, store_id, site_code, goal_name, acos_target, daily_budget_cap,
       risk_profile, auto_approve_enabled, auto_approve_budget_delta_pct, auto_approve_bid_delta_pct,
       status, effective_from, updated_by
FROM agent_goal
WHERE id = ?`, goalID)
	goal, err := scanGoalRow(goalRow)
	if err != nil {
		return models.AdGoal{}, err
	}

	if err := r.insertAuditLogTx(tx, userID, "goal_upsert", "agent_goal", goalID, "success", `{"source":"api"}`); err != nil {
		return models.AdGoal{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.AdGoal{}, err
	}

	return goal, nil
}

func (r *Repository) ListGoals(storeID string) ([]models.AdGoal, error) {
	rows, err := r.db.Query(`
SELECT id, agent_type, store_id, site_code, goal_name, acos_target, daily_budget_cap,
       risk_profile, auto_approve_enabled, auto_approve_budget_delta_pct, auto_approve_bid_delta_pct,
       status, effective_from, updated_by
FROM agent_goal
WHERE store_id = ?
ORDER BY updated_at DESC`, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	goals := make([]models.AdGoal, 0)
	for rows.Next() {
		goal, scanErr := scanGoalRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		goals = append(goals, goal)
	}

	return goals, nil
}

func (r *Repository) CreateGoal(storeID, userID string, input models.UpdateGoalRequest) (models.AdGoal, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return models.AdGoal{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	siteCode := "US"
	_ = tx.QueryRow(`SELECT site_code FROM store WHERE id = ? LIMIT 1`, storeID).Scan(&siteCode)

	_, err = tx.Exec(`
UPDATE agent_goal
SET status = 'paused',
    updated_by = ?,
    updated_at = UTC_TIMESTAMP()
WHERE store_id = ? AND status = 'active'`, userID, storeID)
	if err != nil {
		return models.AdGoal{}, err
	}

	goalID := newID("goal")
	_, err = tx.Exec(`
INSERT INTO agent_goal (
  id, agent_type, store_id, site_code, goal_name, acos_target, daily_budget_cap,
  risk_profile, auto_approve_enabled, auto_approve_budget_delta_pct, auto_approve_bid_delta_pct,
  status, effective_from, updated_by, created_at, updated_at
)
VALUES (?, 'ad_agent', ?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', UTC_TIMESTAMP(), ?, UTC_TIMESTAMP(), UTC_TIMESTAMP())`,
		goalID, storeID, siteCode, input.GoalName, input.ACOSTarget, input.DailyBudgetCap,
		input.RiskProfile, input.AutoApproveEnabled, input.AutoApproveBudgetDeltaPct, input.AutoApproveBidDeltaPct,
		userID)
	if err != nil {
		return models.AdGoal{}, err
	}

	row := tx.QueryRow(`
SELECT id, agent_type, store_id, site_code, goal_name, acos_target, daily_budget_cap,
       risk_profile, auto_approve_enabled, auto_approve_budget_delta_pct, auto_approve_bid_delta_pct,
       status, effective_from, updated_by
FROM agent_goal
WHERE id = ?`, goalID)
	goal, err := scanGoalRow(row)
	if err != nil {
		return models.AdGoal{}, err
	}

	if err := r.insertAuditLogTx(tx, userID, "goal_create", "agent_goal", goalID, "success", `{"source":"api"}`); err != nil {
		return models.AdGoal{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.AdGoal{}, err
	}

	return goal, nil
}

func (r *Repository) UpdateGoalByID(storeID, userID, goalID string, input models.UpdateGoalRequest) (models.AdGoal, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return models.AdGoal{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	result, err := tx.Exec(`
UPDATE agent_goal
SET goal_name = ?,
    acos_target = ?,
    daily_budget_cap = ?,
    risk_profile = ?,
    auto_approve_enabled = ?,
    auto_approve_budget_delta_pct = ?,
    auto_approve_bid_delta_pct = ?,
    effective_from = UTC_TIMESTAMP(),
    updated_by = ?,
    updated_at = UTC_TIMESTAMP()
WHERE id = ? AND store_id = ?`,
		input.GoalName, input.ACOSTarget, input.DailyBudgetCap, input.RiskProfile,
		input.AutoApproveEnabled, input.AutoApproveBudgetDeltaPct, input.AutoApproveBidDeltaPct,
		userID, goalID, storeID)
	if err != nil {
		return models.AdGoal{}, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return models.AdGoal{}, err
	}
	if affected == 0 {
		return models.AdGoal{}, ErrNotFound
	}

	row := tx.QueryRow(`
SELECT id, agent_type, store_id, site_code, goal_name, acos_target, daily_budget_cap,
       risk_profile, auto_approve_enabled, auto_approve_budget_delta_pct, auto_approve_bid_delta_pct,
       status, effective_from, updated_by
FROM agent_goal
WHERE id = ?`, goalID)
	goal, err := scanGoalRow(row)
	if err != nil {
		return models.AdGoal{}, err
	}

	if err := r.insertAuditLogTx(tx, userID, "goal_update", "agent_goal", goalID, "success", `{"source":"api"}`); err != nil {
		return models.AdGoal{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.AdGoal{}, err
	}

	return goal, nil
}

func (r *Repository) DeleteGoalByID(storeID, userID, goalID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	result, err := tx.Exec(`
UPDATE agent_goal
SET status = 'paused',
    updated_by = ?,
    updated_at = UTC_TIMESTAMP()
WHERE id = ? AND store_id = ?`, userID, goalID, storeID)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}

	if err := r.insertAuditLogTx(tx, userID, "goal_delete", "agent_goal", goalID, "success", `{"source":"api"}`); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) ListSuggestions(filter SuggestionFilter) (models.SuggestionsPayload, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	args := make([]any, 0)
	where := []string{"store_id = ?"}
	args = append(args, filter.StoreID)

	if filter.SiteCode != "" {
		where = append(where, "site_code = ?")
		args = append(args, filter.SiteCode)
	}
	if filter.Status != "" {
		where = append(where, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.RiskLevel != "" {
		where = append(where, "risk_level = ?")
		args = append(args, filter.RiskLevel)
	}

	whereClause := strings.Join(where, " AND ")
	countSQL := fmt.Sprintf("SELECT COUNT(1) FROM suggestion WHERE %s", whereClause)
	row := r.db.QueryRow(countSQL, args...)
	var total int
	if err := row.Scan(&total); err != nil {
		return models.SuggestionsPayload{}, err
	}

	querySQL := fmt.Sprintf(`
SELECT id, agent_type, store_id, site_code, goal_id, target_type, target_id, suggestion_type,
       title, reason_summary, risk_level, impact_estimate_json, action_payload_json,
       status, expires_at, created_at
FROM suggestion
WHERE %s
ORDER BY created_at DESC
LIMIT ? OFFSET ?`, whereClause)
	queryArgs := append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)

	rows, err := r.db.Query(querySQL, queryArgs...)
	if err != nil {
		return models.SuggestionsPayload{}, err
	}
	defer rows.Close()

	list := make([]models.Suggestion, 0)
	for rows.Next() {
		suggestion, err := scanSuggestion(rows)
		if err != nil {
			return models.SuggestionsPayload{}, err
		}
		list = append(list, suggestion)
	}

	unreadRow := r.db.QueryRow(`
SELECT COUNT(1)
FROM suggestion
WHERE store_id = ?
  AND risk_level = 'high'
  AND status IN ('ready', 'pending_approval')`, filter.StoreID)
	var unreadHigh int
	if err := unreadRow.Scan(&unreadHigh); err != nil {
		return models.SuggestionsPayload{}, err
	}

	return models.SuggestionsPayload{
		List:                list,
		Total:               total,
		UnreadHighRiskCount: unreadHigh,
	}, nil
}

func (r *Repository) GetSuggestionByID(storeID, suggestionID string) (models.Suggestion, error) {
	row := r.db.QueryRow(`
SELECT id, agent_type, store_id, site_code, goal_id, target_type, target_id, suggestion_type,
       title, reason_summary, risk_level, impact_estimate_json, action_payload_json,
       status, expires_at, created_at
FROM suggestion
WHERE id = ? AND store_id = ?
LIMIT 1`, suggestionID, storeID)

	suggestion, err := scanSuggestion(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Suggestion{}, ErrNotFound
		}
		return models.Suggestion{}, err
	}

	return suggestion, nil
}

func (r *Repository) ApproveSuggestion(storeID, actorID, suggestionID string, input models.ApproveSuggestionRequest) (models.ApproveSuggestionResponse, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return models.ApproveSuggestionResponse{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	response, err := r.approveSuggestionTx(tx, storeID, actorID, suggestionID, input)
	if err != nil {
		return models.ApproveSuggestionResponse{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.ApproveSuggestionResponse{}, err
	}

	return response, nil
}

func (r *Repository) BatchApproveSuggestions(storeID, actorID string, input models.BatchApproveRequest) ([]models.ApproveSuggestionResponse, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	results := make([]models.ApproveSuggestionResponse, 0, len(input.SuggestionIDs))
	for _, suggestionID := range input.SuggestionIDs {
		result, approveErr := r.approveSuggestionTx(tx, storeID, actorID, suggestionID, models.ApproveSuggestionRequest{
			Note:               input.Note,
			ExecuteImmediately: input.ExecuteImmediately,
		})
		if approveErr != nil {
			err = approveErr
			return nil, err
		}
		results = append(results, result)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return results, nil
}

func (r *Repository) RejectSuggestion(storeID, actorID, suggestionID string, input models.RejectSuggestionRequest) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	row := tx.QueryRow(`
SELECT risk_level, status
FROM suggestion
WHERE id = ? AND store_id = ?
LIMIT 1
FOR UPDATE`, suggestionID, storeID)

	var riskLevel string
	var status string
	if err := row.Scan(&riskLevel, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	if err := ValidateSuggestionTransition(status, "rejected"); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidTransition, err)
	}

	if _, err := tx.Exec(`
UPDATE suggestion
SET status = 'rejected', updated_at = UTC_TIMESTAMP()
WHERE id = ?`, suggestionID); err != nil {
		return err
	}

	approvalID := newID("ap")
	_, err = tx.Exec(`
INSERT INTO approval (
  id, suggestion_id, store_id, risk_level, status,
  requested_by, approved_by, decision_note, decided_at, created_at
)
VALUES (?, ?, ?, ?, 'rejected', ?, ?, ?, UTC_TIMESTAMP(), UTC_TIMESTAMP())`,
		approvalID, suggestionID, storeID, riskLevel, actorID, actorID, nullString(input.Note))
	if err != nil {
		return err
	}

	if err := r.insertAuditLogTx(tx, actorID, "suggestion_rejected", "suggestion", suggestionID, "success", fmt.Sprintf(`{"approval_id":"%s"}`, approvalID)); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) ListTasks(filter TaskFilter) ([]models.Task, int, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	args := []any{filter.StoreID}
	where := []string{"store_id = ?"}
	if filter.Status != "" {
		where = append(where, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.RiskLevel != "" {
		where = append(where, "risk_level = ?")
		args = append(args, filter.RiskLevel)
	}

	whereClause := strings.Join(where, " AND ")
	countSQL := fmt.Sprintf("SELECT COUNT(1) FROM task WHERE %s", whereClause)
	row := r.db.QueryRow(countSQL, args...)
	var total int
	if err := row.Scan(&total); err != nil {
		return nil, 0, err
	}

	querySQL := fmt.Sprintf(`
SELECT id, agent_type, suggestion_id, approval_id, task_type, target_type, target_id,
       risk_level, payload_json, status, retry_count, failure_reason, created_by,
       approved_by, executed_at, finished_at, created_at
FROM task
WHERE %s
ORDER BY created_at DESC
LIMIT ? OFFSET ?`, whereClause)
	queryArgs := append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)

	rows, err := r.db.Query(querySQL, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	tasks := make([]models.Task, 0)
	for rows.Next() {
		task, scanErr := scanTask(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		tasks = append(tasks, task)
	}

	return tasks, total, nil
}

func (r *Repository) GetTask(storeID, taskID string) (models.Task, error) {
	row := r.db.QueryRow(`
SELECT id, agent_type, suggestion_id, approval_id, task_type, target_type, target_id,
       risk_level, payload_json, status, retry_count, failure_reason, created_by,
       approved_by, executed_at, finished_at, created_at
FROM task
WHERE id = ? AND store_id = ?
LIMIT 1`, taskID, storeID)

	task, err := scanTask(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Task{}, ErrNotFound
		}
		return models.Task{}, err
	}
	return task, nil
}

func (r *Repository) GetReviewSnapshot(storeID, taskID string) (models.ReviewSnapshot, error) {
	row := r.db.QueryRow(`
SELECT id, agent_type, task_id, store_id, status, before_metrics_json, after_metrics_json, summary, generated_at
FROM review_snapshot
WHERE task_id = ? AND store_id = ?
ORDER BY generated_at DESC
LIMIT 1`, taskID, storeID)

	snapshot, err := scanReviewSnapshot(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.ReviewSnapshot{}, ErrNotFound
		}
		return models.ReviewSnapshot{}, err
	}

	return snapshot, nil
}

func (r *Repository) UpsertReviewSnapshot(storeID, taskID, status string, beforeMetrics, afterMetrics map[string]any, summary string) (models.ReviewSnapshot, error) {
	if beforeMetrics == nil {
		beforeMetrics = map[string]any{}
	}

	beforeRaw, err := json.Marshal(beforeMetrics)
	if err != nil {
		return models.ReviewSnapshot{}, err
	}

	var afterRaw any
	if len(afterMetrics) > 0 {
		encoded, marshalErr := json.Marshal(afterMetrics)
		if marshalErr != nil {
			return models.ReviewSnapshot{}, marshalErr
		}
		afterRaw = encoded
	}

	_, err = r.db.Exec(`
INSERT INTO review_snapshot (
  id, agent_type, task_id, store_id, status, before_metrics_json, after_metrics_json, summary, generated_at
)
VALUES (?, 'ad_agent', ?, ?, ?, ?, ?, ?, UTC_TIMESTAMP())
ON DUPLICATE KEY UPDATE
  status = VALUES(status),
  before_metrics_json = VALUES(before_metrics_json),
  after_metrics_json = VALUES(after_metrics_json),
  summary = VALUES(summary),
  generated_at = UTC_TIMESTAMP()`,
		newID("rvw"), taskID, storeID, status, beforeRaw, afterRaw, nullString(summary))
	if err != nil {
		return models.ReviewSnapshot{}, err
	}

	return r.GetReviewSnapshot(storeID, taskID)
}

func (r *Repository) ListQueuedTasks(limit int, storeID string) ([]QueuedTask, error) {
	if limit <= 0 {
		limit = 20
	}

	where := []string{"status = 'queued'"}
	args := make([]any, 0, 2)
	if strings.TrimSpace(storeID) != "" {
		where = append(where, "store_id = ?")
		args = append(args, storeID)
	}
	whereClause := strings.Join(where, " AND ")

	query := fmt.Sprintf(`
SELECT store_id, id, agent_type, suggestion_id, approval_id, task_type, target_type, target_id,
       risk_level, payload_json, status, retry_count, failure_reason, created_by,
       approved_by, executed_at, finished_at, created_at
FROM task
WHERE %s
ORDER BY created_at ASC
LIMIT ?`, whereClause)
	args = append(args, limit)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]QueuedTask, 0)
	for rows.Next() {
		item, scanErr := scanQueuedTask(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		list = append(list, item)
	}

	return list, nil
}

func (r *Repository) ListTaskEvents(taskID string) ([]models.TaskEvent, error) {
	rows, err := r.db.Query(`
SELECT id, task_id, from_status, to_status, event_type, event_payload_json, created_at
FROM task_event
WHERE task_id = ?
ORDER BY created_at ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]models.TaskEvent, 0)
	for rows.Next() {
		var event models.TaskEvent
		var fromStatus sql.NullString
		var payload sql.NullString
		var createdAt time.Time
		if err := rows.Scan(&event.ID, &event.TaskID, &fromStatus, &event.ToStatus, &event.EventType, &payload, &createdAt); err != nil {
			return nil, err
		}
		if fromStatus.Valid {
			event.FromStatus = &fromStatus.String
		}
		if payload.Valid {
			event.EventPayloadJSON = &payload.String
		}
		event.CreatedAt = toRFC3339(createdAt)
		events = append(events, event)
	}

	return events, nil
}

func (r *Repository) UpdateTaskStatus(storeID, actorID, taskID, nextStatus, reason string) (models.Task, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return models.Task{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	row := tx.QueryRow(`
SELECT id, status, risk_level, retry_count
FROM task
WHERE id = ? AND store_id = ?
LIMIT 1
FOR UPDATE`, taskID, storeID)

	var id string
	var currentStatus string
	var riskLevel string
	var retryCount int
	if err := row.Scan(&id, &currentStatus, &riskLevel, &retryCount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Task{}, ErrNotFound
		}
		return models.Task{}, err
	}

	if nextStatus == "queued" && riskLevel == "high" {
		return models.Task{}, ErrApprovalRequired
	}
	if strings.TrimSpace(actorID) == "" {
		actorID = "system_worker"
	}

	if err := ValidateTaskTransition(currentStatus, nextStatus); err != nil {
		return models.Task{}, fmt.Errorf("%w: %v", ErrInvalidTransition, err)
	}

	retryDelta := 0
	if currentStatus == "failed" && nextStatus == "queued" {
		retryDelta = 1
	}

	executedAtExpr := "executed_at"
	finishedAtExpr := "finished_at"
	switch nextStatus {
	case "running":
		executedAtExpr = "COALESCE(executed_at, UTC_TIMESTAMP())"
		finishedAtExpr = "NULL"
	case "succeeded", "failed", "cancelled":
		finishedAtExpr = "UTC_TIMESTAMP()"
	case "queued":
		if currentStatus == "failed" {
			executedAtExpr = "NULL"
			finishedAtExpr = "NULL"
		}
	}

	failureReason := any(nil)
	if nextStatus == "failed" {
		failureReason = nullString(reason)
	}

	updateSQL := fmt.Sprintf(`
UPDATE task
SET status = ?, retry_count = retry_count + ?, failure_reason = ?, executed_at = %s, finished_at = %s, updated_at = UTC_TIMESTAMP()
WHERE id = ?`, executedAtExpr, finishedAtExpr)
	_, err = tx.Exec(updateSQL, nextStatus, retryDelta, failureReason, taskID)
	if err != nil {
		return models.Task{}, err
	}

	if err := r.insertTaskEventTx(tx, taskID, &currentStatus, nextStatus, "task_status_changed", fmt.Sprintf(`{"reason":"%s"}`, sanitizeJSONString(reason))); err != nil {
		return models.Task{}, err
	}

	auditAction := "task_status_changed"
	if nextStatus == "cancelled" {
		auditAction = "task_cancelled"
	}
	if nextStatus == "queued" && currentStatus == "failed" {
		auditAction = "task_retried"
	}
	if err := r.insertAuditLogTx(tx, actorID, auditAction, "task", taskID, "success", fmt.Sprintf(`{"from_status":"%s","to_status":"%s"}`, currentStatus, nextStatus)); err != nil {
		return models.Task{}, err
	}

	row = tx.QueryRow(`
SELECT id, agent_type, suggestion_id, approval_id, task_type, target_type, target_id,
       risk_level, payload_json, status, retry_count, failure_reason, created_by,
       approved_by, executed_at, finished_at, created_at
FROM task
WHERE id = ?`, taskID)
	task, err := scanTask(row)
	if err != nil {
		return models.Task{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.Task{}, err
	}

	return task, nil
}

func (r *Repository) CreateNotificationForStore(storeID, messageType, priority, title, body string, targetType, targetID *string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	userRows, err := tx.Query(`
SELECT DISTINCT user_id
FROM role_assignment
WHERE store_id = ?`, storeID)
	if err != nil {
		return err
	}
	defer userRows.Close()

	for userRows.Next() {
		var userID string
		if err := userRows.Scan(&userID); err != nil {
			return err
		}

		_, err = tx.Exec(`
INSERT INTO notification (
  id, user_id, agent_type, message_type, priority, title, body,
  target_type, target_id, is_read, created_at
)
VALUES (?, ?, 'ad_agent', ?, ?, ?, ?, ?, ?, 0, UTC_TIMESTAMP())`,
			newID("ntf"), userID, messageType, priority, title, body, targetType, targetID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repository) ListNotifications(userID string, limit int) ([]models.Notification, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := r.db.Query(`
SELECT id, user_id, agent_type, message_type, priority, title, body, target_type, target_id, is_read, created_at
FROM notification
WHERE user_id = ?
ORDER BY created_at DESC
LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]models.Notification, 0)
	for rows.Next() {
		var item models.Notification
		var targetType sql.NullString
		var targetID sql.NullString
		var createdAt time.Time
		if err := rows.Scan(&item.ID, &item.UserID, &item.AgentType, &item.MessageType, &item.Priority,
			&item.Title, &item.Body, &targetType, &targetID, &item.IsRead, &createdAt); err != nil {
			return nil, err
		}
		if targetType.Valid {
			item.TargetType = &targetType.String
		}
		if targetID.Valid {
			item.TargetID = &targetID.String
		}
		item.CreatedAt = toRFC3339(createdAt)
		list = append(list, item)
	}

	return list, nil
}

func (r *Repository) MarkNotificationRead(userID, notificationID string) error {
	result, err := r.db.Exec(`
UPDATE notification
SET is_read = 1
WHERE id = ? AND user_id = ?`, notificationID, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) ListAuditLogs(storeID string, limit int) ([]models.AuditLog, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := r.db.Query(`
SELECT id, agent_type, actor_id, action, target_type, target_id, result, metadata_json, created_at
FROM audit_log
WHERE store_id = ?
ORDER BY created_at DESC
LIMIT ?`, storeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]models.AuditLog, 0)
	for rows.Next() {
		var item models.AuditLog
		var metadata sql.NullString
		var createdAt time.Time
		if err := rows.Scan(&item.ID, &item.AgentType, &item.ActorID, &item.Action,
			&item.TargetType, &item.TargetID, &item.Result, &metadata, &createdAt); err != nil {
			return nil, err
		}
		if metadata.Valid {
			item.MetadataJSON = &metadata.String
		}
		item.CreatedAt = toRFC3339(createdAt)
		list = append(list, item)
	}

	return list, nil
}

func (r *Repository) ListUserIDsByStore(storeID string) ([]string, error) {
	rows, err := r.db.Query(`
SELECT DISTINCT user_id
FROM role_assignment
WHERE store_id = ?`, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		ids = append(ids, userID)
	}

	return ids, nil
}

func (r *Repository) GetLatestApprovalBySuggestion(suggestionID string) (*models.Approval, error) {
	row := r.db.QueryRow(`
SELECT id, suggestion_id, store_id, risk_level, status, requested_by, approved_by, decision_note, decided_at, created_at
FROM approval
WHERE suggestion_id = ?
ORDER BY created_at DESC
LIMIT 1`, suggestionID)

	var item models.Approval
	var approvedBy sql.NullString
	var decisionNote sql.NullString
	var decidedAt sql.NullTime
	var createdAt time.Time
	if err := row.Scan(&item.ID, &item.SuggestionID, &item.StoreID, &item.RiskLevel, &item.Status,
		&item.RequestedBy, &approvedBy, &decisionNote, &decidedAt, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if approvedBy.Valid {
		item.ApprovedBy = &approvedBy.String
	}
	if decisionNote.Valid {
		item.DecisionNote = &decisionNote.String
	}
	if decidedAt.Valid {
		formatted := toRFC3339(decidedAt.Time)
		item.DecidedAt = &formatted
	}
	item.CreatedAt = toRFC3339(createdAt)

	return &item, nil
}

func (r *Repository) approveSuggestionTx(tx *sql.Tx, storeID, actorID, suggestionID string, input models.ApproveSuggestionRequest) (models.ApproveSuggestionResponse, error) {
	row := tx.QueryRow(`
SELECT suggestion_type, target_type, target_id, risk_level, status, action_payload_json
FROM suggestion
WHERE id = ? AND store_id = ?
LIMIT 1
FOR UPDATE`, suggestionID, storeID)

	var suggestionType string
	var targetType string
	var targetID string
	var riskLevel string
	var status string
	var actionPayloadRaw []byte
	if err := row.Scan(&suggestionType, &targetType, &targetID, &riskLevel, &status, &actionPayloadRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.ApproveSuggestionResponse{}, ErrNotFound
		}
		return models.ApproveSuggestionResponse{}, err
	}

	if err := ValidateSuggestionTransition(status, "approved"); err != nil {
		return models.ApproveSuggestionResponse{}, fmt.Errorf("%w: %v", ErrInvalidTransition, err)
	}

	if _, err := tx.Exec(`
UPDATE suggestion
SET status = 'approved', updated_at = UTC_TIMESTAMP()
WHERE id = ?`, suggestionID); err != nil {
		return models.ApproveSuggestionResponse{}, err
	}

	approvalID := newID("ap")
	_, err := tx.Exec(`
INSERT INTO approval (
  id, suggestion_id, store_id, risk_level, status,
  requested_by, approved_by, decision_note, decided_at, created_at
)
VALUES (?, ?, ?, ?, 'approved', ?, ?, ?, UTC_TIMESTAMP(), UTC_TIMESTAMP())`,
		approvalID, suggestionID, storeID, riskLevel, actorID, actorID, nullString(input.Note))
	if err != nil {
		return models.ApproveSuggestionResponse{}, err
	}

	taskStatus := "approved"
	if input.ExecuteImmediately {
		taskStatus = "queued"
	}

	taskID := newID("task")
	_, err = tx.Exec(`
INSERT INTO task (
  id, agent_type, store_id, suggestion_id, approval_id, task_type, target_type, target_id,
  risk_level, payload_json, status, retry_count, failure_reason,
  created_by, approved_by, executed_at, finished_at, created_at, updated_at
)
VALUES (?, 'ad_agent', ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, NULL, ?, ?, NULL, NULL, UTC_TIMESTAMP(), UTC_TIMESTAMP())`,
		taskID, storeID, suggestionID, approvalID, suggestionType, targetType, targetID,
		riskLevel, string(actionPayloadRaw), taskStatus, actorID, actorID)
	if err != nil {
		return models.ApproveSuggestionResponse{}, err
	}

	if err := r.insertTaskEventTx(tx, taskID, nil, taskStatus, "task_created", fmt.Sprintf(`{"suggestion_id":"%s"}`, suggestionID)); err != nil {
		return models.ApproveSuggestionResponse{}, err
	}

	if err := r.insertAuditLogTx(tx, actorID, "suggestion_approved", "suggestion", suggestionID, "success", fmt.Sprintf(`{"approval_id":"%s"}`, approvalID)); err != nil {
		return models.ApproveSuggestionResponse{}, err
	}
	if err := r.insertAuditLogTx(tx, actorID, "task_created", "task", taskID, "success", fmt.Sprintf(`{"task_status":"%s"}`, taskStatus)); err != nil {
		return models.ApproveSuggestionResponse{}, err
	}

	return models.ApproveSuggestionResponse{
		ApprovalID: approvalID,
		TaskID:     taskID,
		TaskStatus: taskStatus,
	}, nil
}

func (r *Repository) insertTaskEventTx(tx *sql.Tx, taskID string, fromStatus *string, toStatus, eventType, payload string) error {
	_, err := tx.Exec(`
INSERT INTO task_event (id, task_id, from_status, to_status, event_type, event_payload_json, created_at)
VALUES (?, ?, ?, ?, ?, ?, UTC_TIMESTAMP())`,
		newID("tev"), taskID, fromStatus, toStatus, eventType, nullString(payload))
	return err
}

func (r *Repository) insertAuditLogTx(tx *sql.Tx, actorID, action, targetType, targetID, result, metadata string) error {
	storeID := ""
	if targetType == "task" {
		_ = tx.QueryRow("SELECT store_id FROM task WHERE id = ?", targetID).Scan(&storeID)
	}
	if targetType == "suggestion" {
		_ = tx.QueryRow("SELECT store_id FROM suggestion WHERE id = ?", targetID).Scan(&storeID)
	}
	if targetType == "agent_goal" {
		_ = tx.QueryRow("SELECT store_id FROM agent_goal WHERE id = ?", targetID).Scan(&storeID)
	}
	if storeID == "" {
		storeID = "store_us_001"
	}

	_, err := tx.Exec(`
INSERT INTO audit_log (id, store_id, agent_type, actor_id, action, target_type, target_id, result, metadata_json, created_at)
VALUES (?, ?, 'ad_agent', ?, ?, ?, ?, ?, ?, UTC_TIMESTAMP())`,
		newID("audit"), storeID, actorID, action, targetType, targetID, result, nullString(metadata))
	return err
}

func scanGoalRow(scanner interface {
	Scan(dest ...any) error
}) (models.AdGoal, error) {
	var goal models.AdGoal
	var acos sql.NullString
	var budget sql.NullString
	var budgetDelta sql.NullString
	var bidDelta sql.NullString
	var effectiveFrom time.Time
	if err := scanner.Scan(&goal.ID, &goal.AgentType, &goal.StoreID, &goal.SiteCode,
		&goal.GoalName, &acos, &budget, &goal.RiskProfile,
		&goal.AutoApproveEnabled, &budgetDelta, &bidDelta,
		&goal.Status, &effectiveFrom, &goal.UpdatedBy); err != nil {
		return models.AdGoal{}, err
	}

	if acos.Valid {
		goal.ACOSTarget = &acos.String
	}
	if budget.Valid {
		goal.DailyBudgetCap = &budget.String
	}
	if budgetDelta.Valid {
		goal.AutoApproveBudgetDeltaPct = &budgetDelta.String
	}
	if bidDelta.Valid {
		goal.AutoApproveBidDeltaPct = &bidDelta.String
	}
	goal.EffectiveFrom = toRFC3339(effectiveFrom)

	return goal, nil
}

func scanSuggestion(scanner interface {
	Scan(dest ...any) error
}) (models.Suggestion, error) {
	var item models.Suggestion
	var impactRaw []byte
	var actionRaw []byte
	var expiresAt sql.NullTime
	var createdAt time.Time

	if err := scanner.Scan(&item.ID, &item.AgentType, &item.StoreID, &item.SiteCode,
		&item.GoalID, &item.TargetType, &item.TargetID, &item.SuggestionType,
		&item.Title, &item.ReasonSummary, &item.RiskLevel, &impactRaw, &actionRaw,
		&item.Status, &expiresAt, &createdAt); err != nil {
		return models.Suggestion{}, err
	}

	if len(impactRaw) > 0 {
		item.ImpactEstimateJSON = decodeJSONMap(impactRaw)
	}
	item.ActionPayloadJSON = decodeJSONMap(actionRaw)
	if expiresAt.Valid {
		expires := toRFC3339(expiresAt.Time)
		item.ExpiresAt = &expires
	}
	item.CreatedAt = toRFC3339(createdAt)

	return item, nil
}

func scanTask(scanner interface {
	Scan(dest ...any) error
}) (models.Task, error) {
	var item models.Task
	var approvalID sql.NullString
	var failureReason sql.NullString
	var approvedBy sql.NullString
	var executedAt sql.NullTime
	var finishedAt sql.NullTime
	var createdAt time.Time

	if err := scanner.Scan(&item.ID, &item.AgentType, &item.SuggestionID, &approvalID,
		&item.TaskType, &item.TargetType, &item.TargetID, &item.RiskLevel,
		&item.PayloadJSON, &item.Status, &item.RetryCount, &failureReason,
		&item.CreatedBy, &approvedBy, &executedAt, &finishedAt, &createdAt); err != nil {
		return models.Task{}, err
	}

	if approvalID.Valid {
		item.ApprovalID = &approvalID.String
	}
	if failureReason.Valid {
		item.FailureReason = &failureReason.String
	}
	if approvedBy.Valid {
		item.ApprovedBy = &approvedBy.String
	}
	if executedAt.Valid {
		execStr := toRFC3339(executedAt.Time)
		item.ExecutedAt = &execStr
	}
	if finishedAt.Valid {
		finishStr := toRFC3339(finishedAt.Time)
		item.FinishedAt = &finishStr
	}
	item.CreatedAt = toRFC3339(createdAt)

	return item, nil
}

func scanQueuedTask(scanner interface {
	Scan(dest ...any) error
}) (QueuedTask, error) {
	var item QueuedTask
	var approvalID sql.NullString
	var failureReason sql.NullString
	var approvedBy sql.NullString
	var executedAt sql.NullTime
	var finishedAt sql.NullTime
	var createdAt time.Time

	if err := scanner.Scan(&item.StoreID, &item.Task.ID, &item.Task.AgentType, &item.Task.SuggestionID, &approvalID,
		&item.Task.TaskType, &item.Task.TargetType, &item.Task.TargetID, &item.Task.RiskLevel,
		&item.Task.PayloadJSON, &item.Task.Status, &item.Task.RetryCount, &failureReason,
		&item.Task.CreatedBy, &approvedBy, &executedAt, &finishedAt, &createdAt); err != nil {
		return QueuedTask{}, err
	}

	if approvalID.Valid {
		item.Task.ApprovalID = &approvalID.String
	}
	if failureReason.Valid {
		item.Task.FailureReason = &failureReason.String
	}
	if approvedBy.Valid {
		item.Task.ApprovedBy = &approvedBy.String
	}
	if executedAt.Valid {
		execStr := toRFC3339(executedAt.Time)
		item.Task.ExecutedAt = &execStr
	}
	if finishedAt.Valid {
		finishStr := toRFC3339(finishedAt.Time)
		item.Task.FinishedAt = &finishStr
	}
	item.Task.CreatedAt = toRFC3339(createdAt)

	return item, nil
}

func decodeJSONMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	if out == nil {
		return map[string]any{}
	}
	return out
}

func scanReviewSnapshot(scanner interface {
	Scan(dest ...any) error
}) (models.ReviewSnapshot, error) {
	var item models.ReviewSnapshot
	var beforeRaw []byte
	var afterRaw []byte
	var summary sql.NullString
	var generatedAt time.Time

	if err := scanner.Scan(&item.ID, &item.AgentType, &item.TaskID, &item.StoreID, &item.Status, &beforeRaw, &afterRaw, &summary, &generatedAt); err != nil {
		return models.ReviewSnapshot{}, err
	}

	item.BeforeMetrics = decodeJSONMap(beforeRaw)
	if len(afterRaw) > 0 && string(afterRaw) != "null" {
		item.AfterMetrics = decodeJSONMap(afterRaw)
	}
	if summary.Valid {
		item.Summary = &summary.String
	}
	item.GeneratedAt = toRFC3339(generatedAt)

	return item, nil
}

func nullString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func sanitizeJSONString(value string) string {
	replacer := strings.NewReplacer(`\\`, `\\\\`, `"`, `\\"`, "\n", " ", "\r", " ", "\t", " ")
	return replacer.Replace(value)
}
