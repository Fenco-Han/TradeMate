CREATE TABLE IF NOT EXISTS review_snapshot (
  id VARCHAR(64) PRIMARY KEY,
  agent_type VARCHAR(32) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  store_id VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  before_metrics_json JSON NOT NULL,
  after_metrics_json JSON NULL,
  summary TEXT NULL,
  generated_at DATETIME NOT NULL,
  UNIQUE KEY uk_review_task_id (task_id),
  INDEX idx_review_store_generated (store_id, generated_at)
);
