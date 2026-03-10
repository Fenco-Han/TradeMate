ALTER TABLE user_account
  ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255) NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS approval (
  id VARCHAR(64) PRIMARY KEY,
  suggestion_id VARCHAR(64) NOT NULL,
  store_id VARCHAR(64) NOT NULL,
  risk_level VARCHAR(16) NOT NULL,
  status VARCHAR(32) NOT NULL,
  requested_by VARCHAR(64) NOT NULL,
  approved_by VARCHAR(64) NULL,
  decision_note TEXT NULL,
  decided_at DATETIME NULL,
  created_at DATETIME NOT NULL,
  INDEX idx_approval_suggestion_id (suggestion_id),
  INDEX idx_approval_store_id (store_id)
);

CREATE TABLE IF NOT EXISTS task (
  id VARCHAR(64) PRIMARY KEY,
  agent_type VARCHAR(32) NOT NULL,
  store_id VARCHAR(64) NOT NULL,
  suggestion_id VARCHAR(64) NOT NULL,
  approval_id VARCHAR(64) NULL,
  task_type VARCHAR(64) NOT NULL,
  target_type VARCHAR(32) NOT NULL,
  target_id VARCHAR(128) NOT NULL,
  risk_level VARCHAR(16) NOT NULL,
  payload_json JSON NOT NULL,
  status VARCHAR(32) NOT NULL,
  retry_count INT NOT NULL DEFAULT 0,
  failure_reason TEXT NULL,
  created_by VARCHAR(64) NOT NULL,
  approved_by VARCHAR(64) NULL,
  executed_at DATETIME NULL,
  finished_at DATETIME NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  INDEX idx_task_store_id (store_id),
  INDEX idx_task_status (status),
  INDEX idx_task_suggestion_id (suggestion_id)
);

CREATE TABLE IF NOT EXISTS task_event (
  id VARCHAR(64) PRIMARY KEY,
  task_id VARCHAR(64) NOT NULL,
  from_status VARCHAR(32) NULL,
  to_status VARCHAR(32) NOT NULL,
  event_type VARCHAR(64) NOT NULL,
  event_payload_json JSON NULL,
  created_at DATETIME NOT NULL,
  INDEX idx_task_event_task_id (task_id)
);

CREATE TABLE IF NOT EXISTS notification (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  agent_type VARCHAR(32) NOT NULL,
  message_type VARCHAR(64) NOT NULL,
  priority VARCHAR(16) NOT NULL,
  title VARCHAR(255) NOT NULL,
  body TEXT NOT NULL,
  target_type VARCHAR(32) NULL,
  target_id VARCHAR(128) NULL,
  is_read TINYINT(1) NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL,
  INDEX idx_notification_user_id (user_id),
  INDEX idx_notification_is_read (is_read)
);

CREATE TABLE IF NOT EXISTS audit_log (
  id VARCHAR(64) PRIMARY KEY,
  store_id VARCHAR(64) NOT NULL,
  agent_type VARCHAR(32) NOT NULL,
  actor_id VARCHAR(64) NOT NULL,
  action VARCHAR(64) NOT NULL,
  target_type VARCHAR(32) NOT NULL,
  target_id VARCHAR(128) NOT NULL,
  result VARCHAR(32) NOT NULL,
  metadata_json JSON NULL,
  created_at DATETIME NOT NULL,
  INDEX idx_audit_store_id (store_id),
  INDEX idx_audit_target (target_type, target_id)
);
