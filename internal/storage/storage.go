package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB represents the database layer
type DB struct {
	db *sql.DB
}

// APIKey represents an API key with its status
type APIKey struct {
	ID          int       `json:"id"`
	Key         string    `json:"key"`
	QuotaLimit  int       `json:"quota_limit"`
	QuotaUsed   int       `json:"quota_used"`
	IsActive    bool      `json:"is_active"`
	LastUsed    time.Time `json:"last_used"`
	LastChecked time.Time `json:"last_checked"`
	ErrorCount  int       `json:"error_count"`
	CreatedAt   time.Time `json:"created_at"`
	RefreshesAt time.Time `json:"refreshes_at"` // When quota refreshes
}

// RequestLog represents a log entry for an API request
type RequestLog struct {
	ID         int       `json:"id"`
	Path       string    `json:"path"`
	Method     string    `json:"method"`
	StatusCode int       `json:"status_code"`
	KeyID      int       `json:"key_id"`
	Timestamp  time.Time `json:"timestamp"`
}

// New creates a new database connection
func New(dbPath string) (*DB, error) {
	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Check connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize database schema
	if err := initSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &DB{db: db}, nil
}

// Close closes the database connection
func (d *DB) Close() error {
	return d.db.Close()
}

// initSchema initializes the database schema
func initSchema(db *sql.DB) error {
	// Create API keys table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT UNIQUE NOT NULL,
			quota_limit INTEGER DEFAULT 0,
			quota_used INTEGER DEFAULT 0,
			is_active BOOLEAN DEFAULT TRUE,
			last_used TIMESTAMP,
			last_checked TIMESTAMP,
			error_count INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			refreshes_at TIMESTAMP
		);
	`)
	if err != nil {
		return err
	}

	// Create requests log table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS request_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT NOT NULL,
			method TEXT NOT NULL,
			status_code INTEGER,
			key_id INTEGER,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (key_id) REFERENCES api_keys (id)
		);
	`)
	return err
}

// AddAPIKey adds a new API key to the database
func (d *DB) AddAPIKey(key string, quotaLimit int, refreshesAt time.Time) (int, error) {
	result, err := d.db.Exec(
		"INSERT INTO api_keys (key, quota_limit, quota_used, is_active, refreshes_at) VALUES (?, ?, 0, TRUE, ?)",
		key, quotaLimit, refreshesAt,
	)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

// GetAPIKey gets an API key by ID
func (d *DB) GetAPIKey(id int) (*APIKey, error) {
	var key APIKey
	var lastUsed, lastChecked, refreshesAt sql.NullTime

	err := d.db.QueryRow(`
		SELECT id, key, quota_limit, quota_used, is_active, 
		       last_used, last_checked, error_count,
		       created_at, refreshes_at
		FROM api_keys
		WHERE id = ?
	`, id).Scan(
		&key.ID, &key.Key, &key.QuotaLimit, &key.QuotaUsed, &key.IsActive,
		&lastUsed, &lastChecked, &key.ErrorCount,
		&key.CreatedAt, &refreshesAt,
	)

	if err != nil {
		return nil, err
	}

	if lastUsed.Valid {
		key.LastUsed = lastUsed.Time
	}

	if lastChecked.Valid {
		key.LastChecked = lastChecked.Time
	}

	if refreshesAt.Valid {
		key.RefreshesAt = refreshesAt.Time
	}

	return &key, nil
}

// GetAllAPIKeys gets all API keys
func (d *DB) GetAllAPIKeys() ([]*APIKey, error) {
	rows, err := d.db.Query(`
		SELECT id, key, quota_limit, quota_used, is_active, 
		       last_used, last_checked, error_count,
		       created_at, refreshes_at
		FROM api_keys
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*APIKey
	for rows.Next() {
		var key APIKey
		var lastUsed, lastChecked, refreshesAt sql.NullTime

		err := rows.Scan(
			&key.ID, &key.Key, &key.QuotaLimit, &key.QuotaUsed, &key.IsActive,
			&lastUsed, &lastChecked, &key.ErrorCount,
			&key.CreatedAt, &refreshesAt,
		)
		if err != nil {
			return nil, err
		}

		if lastUsed.Valid {
			key.LastUsed = lastUsed.Time
		}

		if lastChecked.Valid {
			key.LastChecked = lastChecked.Time
		}

		if refreshesAt.Valid {
			key.RefreshesAt = refreshesAt.Time
		}

		keys = append(keys, &key)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return keys, nil
}

// GetAvailableAPIKey gets an API key with available quota
func (d *DB) GetAvailableAPIKey() (*APIKey, error) {
	var key APIKey
	var lastUsed, lastChecked, refreshesAt sql.NullTime

	// Try to get a key with available quota
	err := d.db.QueryRow(`
		SELECT id, key, quota_limit, quota_used, is_active, 
		       last_used, last_checked, error_count,
		       created_at, refreshes_at
		FROM api_keys
		WHERE is_active = TRUE AND (quota_limit = 0 OR quota_used < quota_limit)
		ORDER BY quota_used * 1.0 / CASE WHEN quota_limit = 0 THEN 1 ELSE quota_limit END ASC,
		         last_used ASC
		LIMIT 1
	`).Scan(
		&key.ID, &key.Key, &key.QuotaLimit, &key.QuotaUsed, &key.IsActive,
		&lastUsed, &lastChecked, &key.ErrorCount,
		&key.CreatedAt, &refreshesAt,
	)

	if err != nil {
		return nil, err
	}

	if lastUsed.Valid {
		key.LastUsed = lastUsed.Time
	}

	if lastChecked.Valid {
		key.LastChecked = lastChecked.Time
	}

	if refreshesAt.Valid {
		key.RefreshesAt = refreshesAt.Time
	}

	// Check if quota should be reset
	currentTime := time.Now()
	if key.RefreshesAt.Before(currentTime) && !key.RefreshesAt.IsZero() {
		// Calculate next refresh time (default 1st of every month)
		// Use UTC to avoid some potential issues
		nextRefresh := time.Date(
			currentTime.Year(), currentTime.Month(), 1, 0, 0, 0, 0, time.UTC,
		).AddDate(0, 1, 0)

		// Reset quota and update refresh time
		_, err := d.db.Exec(
			"UPDATE api_keys SET quota_used = 0, refreshes_at = ? WHERE id = ?",
			nextRefresh, key.ID,
		)
		if err != nil {
			return nil, err
		}

		key.QuotaUsed = 0
		key.RefreshesAt = nextRefresh
	}

	return &key, nil
}

// IncrementAPIKeyUsage increments the quota used by an API key
func (d *DB) IncrementAPIKeyUsage(id int, incrementQuota int) error {
	_, err := d.db.Exec(
		"UPDATE api_keys SET quota_used = quota_used + ?, last_used = CURRENT_TIMESTAMP WHERE id = ?",
		incrementQuota, id,
	)
	return err
}

// UpdateAPIKeyUsage updates the quota used by an API key
func (d *DB) UpdateAPIKeyUsage(id int, quotaUsed int) error {
	_, err := d.db.Exec(
		"UPDATE api_keys SET quota_used = ?, last_used = CURRENT_TIMESTAMP WHERE id = ?",
		quotaUsed, id,
	)
	return err
}

// UpdateAPIKeyStatus updates the status of an API key
func (d *DB) UpdateAPIKeyStatus(id int, isActive bool, errorCount int) error {
	_, err := d.db.Exec(
		"UPDATE api_keys SET is_active = ?, error_count = ?, last_checked = CURRENT_TIMESTAMP WHERE id = ?",
		isActive, errorCount, id,
	)
	return err
}

// LogRequest logs an API request
func (d *DB) LogRequest(path, method string, statusCode int, keyID int) error {
	_, err := d.db.Exec(
		"INSERT INTO request_log (path, method, status_code, key_id) VALUES (?, ?, ?, ?)",
		path, method, statusCode, keyID,
	)
	return err
}

// DeleteAPIKey deletes an API key
func (d *DB) DeleteAPIKey(id int) error {
	_, err := d.db.Exec("DELETE FROM api_keys WHERE id = ?", id)
	return err
}
