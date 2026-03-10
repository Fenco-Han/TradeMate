-- TradeMate V1 schema draft
-- Scope: platform base + ad agent v1

CREATE TABLE user_account (
  id VARCHAR(64) PRIMARY KEY,
  email VARCHAR(255) NULL,
  phone VARCHAR(64) NULL,
  display_name VARCHAR(128) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE KEY uk_user_email (email),
  UNIQUE KEY uk_user_phone (phone)
);

CREATE TABLE store (
  id VARCHAR(64) PRIMARY KEY,
  site_code VARCHAR(16) NOT NULL,
  store_name VARCHAR(255) NOT NULL,
  currency VARCHAR(16) NOT NULL,
  timezone VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  KEY idx_store_site_status (site_code, status)
);

CREATE TABLE role_assignment (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  store_id VARCHAR(64) NOT NULL,
  role_code VARCHAR(32) NOT NULL,
  created_at DATETIME NOT NULL,
  UNIQUE KEY uk_role_user_store (user_id, store_id, role_code),
  KEY idx_role_store (store_id),
  CONSTRAINT fk_role_user FOREIGN KEY (user_id) REFERENCES user_account(id),
  CONSTRAINT fk_role_store FOREIGN KEY (store_id) REFERENCES store(id)
);

CREATE TABLE ad_account (
  id VARCHAR(64) PRIMARY KEY,
  store_id VARCHAR(64) NOT NULL,
  account_name VARCHAR(255) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  last_sync_at DATETIME NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  KEY idx_ad_account_store (store_id),
  CONSTRAINT fk_ad_account_store FOREIGN KEY (store_id) REFERENCES store(id)
);

CREATE TABLE agent_goal (
  id VARCHAR(64) PRIMARY KEY,
  agent_type VARCHAR(32) NOT NULL,
  store_id VARCHAR(64) NOT NULL,
  site_code VARCHAR(16) NOT NULL,
  goal_name VARCHAR(255) NOT NULL,
  acos_target DECIMAL(10,2) NULL,
  daily_budget_cap DECIMAL(12,2) NULL,
  risk_profile VARCHAR(32) NOT NULL,
  auto_approve_enabled TINYINT(1) NOT NULL DEFAULT 0,
  auto_approve_budget_delta_pct DECIMAL(10,2) NULL,
  auto_approve_bid_delta_pct DECIMAL(10,2) NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  effective_from DATETIME NOT NULL,
  updated_by VARCHAR(64) NOT NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  KEY idx_goal_agent_store_status (agent_type, store_id, status),
  CONSTRAINT fk_goal_store FOREIGN KEY (store_id) REFERENCES store(id),
  CONSTRAINT fk_goal_user FOREIGN KEY (updated_by) REFERENCES user_account(id)
);

CREATE TABLE context_snapshot (
  id VARCHAR(64) PRIMARY KEY,
  agent_type VARCHAR(32) NOT NULL,
  store_id VARCHAR(64) NOT NULL,
  target_type VARCHAR(32) NOT NULL,
  target_id VARCHAR(128) NOT NULL,
  snapshot_date DATE NOT NULL,
  metrics_json JSON NOT NULL,
  source_version VARCHAR(64) NOT NULL,
  created_at DATETIME NOT NULL,
  KEY idx_context_store_target_date (store_id, target_type, target_id, snapshot_date),
  KEY idx_context_agent_date (agent_type, snapshot_date),
  CONSTRAINT fk_context_store FOREIGN KEY (store_id) REFERENCES store(id)
);

CREATE TABLE suggestion (
  id VARCHAR(64) PRIMARY KEY,
  agent_type VARCHAR(32) NOT NULL,
  store_id VARCHAR(64) NOT NULL,
  site_code VARCHAR(16) NOT NULL,
  goal_id VARCHAR(64) NOT NULL,
  target_type VARCHAR(32) NOT NULL,
  target_id VARCHAR(128) NOT NULL,
  suggestion_type VARCHAR(64) NOT NULL,
  title VARCHAR(255) NOT NULL,
  reason_summary TEXT NOT NULL,
  risk_level VARCHAR(16) NOT NULL,
  impact_estimate_json JSON NULL,
  action_payload_json JSON NOT NULL,
  status VARCHAR(32) NOT NULL,
  expires_at DATETIME NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  KEY idx_suggestion_store_status_risk (store_id, status, risk_level, created_at),
  KEY idx_suggestion_target (target_type, target_id, created_at),
  CONSTRAINT fk_suggestion_store FOREIGN KEY (store_id) REFERENCES store(id),
  CONSTRAINT fk_suggestion_goal FOREIGN KEY (goal_id) REFERENCES agent_goal(id)
);

CREATE TABLE approval (
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
  updated_at DATETIME NOT NULL,
  KEY idx_approval_store_status (store_id, status, created_at),
  KEY idx_approval_suggestion (suggestion_id),
  CONSTRAINT fk_approval_suggestion FOREIGN KEY (suggestion_id) REFERENCES suggestion(id),
  CONSTRAINT fk_approval_store FOREIGN KEY (store_id) REFERENCES store(id),
  CONSTRAINT fk_approval_requested_by FOREIGN KEY (requested_by) REFERENCES user_account(id),
  CONSTRAINT fk_approval_approved_by FOREIGN KEY (approved_by) REFERENCES user_account(id)
);

CREATE TABLE task (
  id VARCHAR(64) PRIMARY KEY,
  agent_type VARCHAR(32) NOT NULL,
  suggestion_id VARCHAR(64) NOT NULL,
  approval_id VARCHAR(64) NULL,
  task_type VARCHAR(64) NOT NULL,
  target_type VARCHAR(32) NOT NULL,
  target_id VARCHAR(128) NOT NULL,
  risk_level VARCHAR(16) NOT NULL,
  execution_channel VARCHAR(32) NOT NULL,
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
  KEY idx_task_agent_store_status (agent_type, status, created_at),
  KEY idx_task_target (target_type, target_id, created_at),
  CONSTRAINT fk_task_suggestion FOREIGN KEY (suggestion_id) REFERENCES suggestion(id),
  CONSTRAINT fk_task_approval FOREIGN KEY (approval_id) REFERENCES approval(id),
  CONSTRAINT fk_task_created_by FOREIGN KEY (created_by) REFERENCES user_account(id),
  CONSTRAINT fk_task_approved_by FOREIGN KEY (approved_by) REFERENCES user_account(id)
);

CREATE TABLE task_event (
  id VARCHAR(64) PRIMARY KEY,
  task_id VARCHAR(64) NOT NULL,
  from_status VARCHAR(32) NULL,
  to_status VARCHAR(32) NOT NULL,
  event_type VARCHAR(64) NOT NULL,
  event_payload_json JSON NULL,
  created_at DATETIME NOT NULL,
  KEY idx_task_event_task_created (task_id, created_at),
  CONSTRAINT fk_task_event_task FOREIGN KEY (task_id) REFERENCES task(id)
);

CREATE TABLE review_snapshot (
  id VARCHAR(64) PRIMARY KEY,
  agent_type VARCHAR(32) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  store_id VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  before_metrics_json JSON NOT NULL,
  after_metrics_json JSON NULL,
  summary TEXT NULL,
  generated_at DATETIME NOT NULL,
  KEY idx_review_task (task_id),
  KEY idx_review_store_generated (store_id, generated_at),
  CONSTRAINT fk_review_task FOREIGN KEY (task_id) REFERENCES task(id),
  CONSTRAINT fk_review_store FOREIGN KEY (store_id) REFERENCES store(id)
);

CREATE TABLE notification (
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
  KEY idx_notification_user_read_created (user_id, is_read, created_at),
  CONSTRAINT fk_notification_user FOREIGN KEY (user_id) REFERENCES user_account(id)
);

CREATE TABLE audit_log (
  id VARCHAR(64) PRIMARY KEY,
  agent_type VARCHAR(32) NOT NULL,
  actor_id VARCHAR(64) NOT NULL,
  action VARCHAR(64) NOT NULL,
  target_type VARCHAR(32) NOT NULL,
  target_id VARCHAR(128) NOT NULL,
  result VARCHAR(32) NOT NULL,
  metadata_json JSON NULL,
  created_at DATETIME NOT NULL,
  KEY idx_audit_actor_created (actor_id, created_at),
  KEY idx_audit_target_created (target_type, target_id, created_at),
  KEY idx_audit_agent_created (agent_type, created_at),
  CONSTRAINT fk_audit_actor FOREIGN KEY (actor_id) REFERENCES user_account(id)
);
