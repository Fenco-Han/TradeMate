CREATE TABLE IF NOT EXISTS user_account (
  id VARCHAR(64) PRIMARY KEY,
  email VARCHAR(255) NULL,
  phone VARCHAR(64) NULL,
  display_name VARCHAR(128) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS store (
  id VARCHAR(64) PRIMARY KEY,
  site_code VARCHAR(16) NOT NULL,
  store_name VARCHAR(255) NOT NULL,
  currency VARCHAR(16) NOT NULL,
  timezone VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS role_assignment (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  store_id VARCHAR(64) NOT NULL,
  role_code VARCHAR(32) NOT NULL,
  created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS agent_goal (
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
  updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS suggestion (
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
  updated_at DATETIME NOT NULL
);
