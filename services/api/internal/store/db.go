package store

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func OpenDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(10 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func ApplyMigrations(db *sql.DB, currentDir string) error {
	dir, err := resolveMigrationsDir(currentDir)
	if err != nil {
		return err
	}

	entries := make([]string, 0)
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".sql") {
			entries = append(entries, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	sort.Strings(entries)
	for _, entry := range entries {
		if err := executeSQLFile(db, entry); err != nil {
			return fmt.Errorf("apply migration %s: %w", entry, err)
		}
	}

	return nil
}

func resolveMigrationsDir(currentDir string) (string, error) {
	candidates := []string{
		filepath.Join(currentDir, "migrations"),
		filepath.Join(currentDir, "services", "api", "migrations"),
		filepath.Join("migrations"),
		filepath.Join("services", "api", "migrations"),
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
	}

	return "", errors.New("migrations directory not found")
}

func executeSQLFile(db *sql.DB, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	statements := make([]string, 0)
	var builder strings.Builder

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}

		builder.WriteString(line)
		builder.WriteRune('\n')
		if strings.HasSuffix(line, ";") {
			stmt := strings.TrimSpace(builder.String())
			stmt = strings.TrimSuffix(stmt, ";")
			if stmt != "" {
				statements = append(statements, stmt)
			}
			builder.Reset()
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(builder.String()) != "" {
		statements = append(statements, strings.TrimSpace(builder.String()))
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}

func SeedDemoData(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.Exec(`
INSERT INTO user_account (id, email, phone, display_name, password_hash, status, created_at, updated_at)
VALUES ('u_demo', 'demo@trademate.dev', NULL, 'Demo User', 'demo123', 'active', UTC_TIMESTAMP(), UTC_TIMESTAMP())
ON DUPLICATE KEY UPDATE display_name = VALUES(display_name), password_hash = VALUES(password_hash), updated_at = UTC_TIMESTAMP()`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
INSERT INTO store (id, site_code, store_name, currency, timezone, status, created_at, updated_at)
VALUES ('store_us_001', 'US', 'TradeMate Demo Store', 'USD', 'America/Los_Angeles', 'active', UTC_TIMESTAMP(), UTC_TIMESTAMP())
ON DUPLICATE KEY UPDATE store_name = VALUES(store_name), updated_at = UTC_TIMESTAMP()`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
INSERT INTO role_assignment (id, user_id, store_id, role_code, created_at)
VALUES ('role_001', 'u_demo', 'store_us_001', 'owner', UTC_TIMESTAMP())
ON DUPLICATE KEY UPDATE role_code = VALUES(role_code)`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
INSERT INTO agent_goal (
  id, agent_type, store_id, site_code, goal_name, acos_target, daily_budget_cap,
  risk_profile, auto_approve_enabled, auto_approve_budget_delta_pct, auto_approve_bid_delta_pct,
  status, effective_from, updated_by, created_at, updated_at
)
VALUES (
  'goal_001', 'ad_agent', 'store_us_001', 'US', 'Default ACOS Goal', 28.00, 800.00,
  'balanced', 1, 10.00, 8.00,
  'active', UTC_TIMESTAMP(), 'u_demo', UTC_TIMESTAMP(), UTC_TIMESTAMP()
)
ON DUPLICATE KEY UPDATE goal_name = VALUES(goal_name), updated_at = UTC_TIMESTAMP()`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
INSERT INTO suggestion (
  id, agent_type, store_id, site_code, goal_id, target_type, target_id,
  suggestion_type, title, reason_summary, risk_level, impact_estimate_json,
  action_payload_json, status, expires_at, created_at, updated_at
)
VALUES
(
  'sg_001', 'ad_agent', 'store_us_001', 'US', 'goal_001', 'campaign', 'cmp_001',
  'budget_increase', 'Increase campaign budget by 20%',
  'Budget utilization stayed above 92% for 3 days and ACOS is below target.',
  'medium', JSON_OBJECT('expected_sales_lift_pct', 8),
  JSON_OBJECT('task_type', 'budget_increase', 'target_type', 'campaign', 'target_id', 'cmp_001', 'before_budget', '50.00', 'after_budget', '60.00'),
  'ready', NULL, UTC_TIMESTAMP(), UTC_TIMESTAMP()
),
(
  'sg_002', 'ad_agent', 'store_us_001', 'US', 'goal_001', 'keyword', 'kw_101',
  'bid_increase', 'Increase keyword bid on high-converting term',
  'Conversion rate is above account median and CPC is within risk threshold.',
  'low', NULL,
  JSON_OBJECT('task_type', 'bid_increase', 'target_type', 'keyword', 'target_id', 'kw_101', 'before_bid', '0.95', 'after_bid', '1.05'),
  'ready', NULL, UTC_TIMESTAMP(), UTC_TIMESTAMP()
)
ON DUPLICATE KEY UPDATE updated_at = UTC_TIMESTAMP()`)
	if err != nil {
		return err
	}

	return tx.Commit()
}
