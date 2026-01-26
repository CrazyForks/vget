package server

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/guiyumin/vget/internal/core/config"
	_ "modernc.org/sqlite"
)

const historyDBFile = "history.db"

// HistoryRecord represents a completed download in history
type HistoryRecord struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	Filename    string `json:"filename"`
	Status      string `json:"status"` // "completed" or "failed"
	SizeBytes   int64  `json:"size_bytes"`
	StartedAt   int64  `json:"started_at"`   // Unix timestamp
	CompletedAt int64  `json:"completed_at"` // Unix timestamp
	Duration    int64  `json:"duration_seconds"`
	Error       string `json:"error,omitempty"`
}

// HistoryDB manages SQLite database for download history
type HistoryDB struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewHistoryDB creates and initializes the history database
func NewHistoryDB() (*HistoryDB, error) {
	// Get config directory
	configDir, err := config.ConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config dir: %w", err)
	}

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config dir: %w", err)
	}

	dbPath := filepath.Join(configDir, historyDBFile)

	// Open database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open history database: %w", err)
	}

	// Create table if not exists (using INTEGER for timestamps)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS download_history (
			id TEXT PRIMARY KEY,
			url TEXT NOT NULL,
			filename TEXT,
			status TEXT NOT NULL,
			size_bytes INTEGER DEFAULT 0,
			started_at INTEGER NOT NULL,
			completed_at INTEGER NOT NULL,
			duration_seconds INTEGER DEFAULT 0,
			error_message TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_completed_at ON download_history(completed_at DESC);
		CREATE INDEX IF NOT EXISTS idx_status ON download_history(status);
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create history table: %w", err)
	}

	return &HistoryDB{db: db}, nil
}

// Close closes the database connection
func (h *HistoryDB) Close() error {
	if h.db != nil {
		return h.db.Close()
	}
	return nil
}

// RecordJob saves a completed or failed job to history
func (h *HistoryDB) RecordJob(job *Job) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	duration := int64(job.UpdatedAt.Sub(job.CreatedAt).Seconds())

	_, err := h.db.Exec(`
		INSERT OR REPLACE INTO download_history
		(id, url, filename, status, size_bytes, started_at, completed_at, duration_seconds, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		job.ID,
		job.URL,
		job.Filename,
		string(job.Status),
		job.Total,
		job.CreatedAt.Unix(),
		job.UpdatedAt.Unix(),
		duration,
		job.Error,
	)

	return err
}

// GetHistory returns download history with pagination
func (h *HistoryDB) GetHistory(limit, offset int) ([]HistoryRecord, int, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Get total count
	var total int
	err := h.db.QueryRow("SELECT COUNT(*) FROM download_history").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count history: %w", err)
	}

	// Get records
	rows, err := h.db.Query(`
		SELECT id, url, filename, status, size_bytes, started_at, completed_at, duration_seconds, error_message
		FROM download_history
		ORDER BY completed_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query history: %w", err)
	}
	defer rows.Close()

	records := make([]HistoryRecord, 0)
	for rows.Next() {
		var r HistoryRecord
		var errorMsg sql.NullString
		var startedAt, completedAt int64

		err := rows.Scan(
			&r.ID,
			&r.URL,
			&r.Filename,
			&r.Status,
			&r.SizeBytes,
			&startedAt,
			&completedAt,
			&r.Duration,
			&errorMsg,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan history row: %w", err)
		}

		r.StartedAt = startedAt
		r.CompletedAt = completedAt
		if errorMsg.Valid {
			r.Error = errorMsg.String
		}
		records = append(records, r)
	}

	return records, total, nil
}

// GetStats returns download statistics
func (h *HistoryDB) GetStats() (completed int, failed int, totalBytes int64, err error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	err = h.db.QueryRow(`
		SELECT
			COUNT(CASE WHEN status = 'completed' THEN 1 END),
			COUNT(CASE WHEN status = 'failed' THEN 1 END),
			COALESCE(SUM(CASE WHEN status = 'completed' THEN size_bytes ELSE 0 END), 0)
		FROM download_history
	`).Scan(&completed, &failed, &totalBytes)

	return
}

// DeleteRecord deletes a single history record
func (h *HistoryDB) DeleteRecord(id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	result, err := h.db.Exec("DELETE FROM download_history WHERE id = ?", id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("record not found")
	}

	return nil
}

// ClearHistory deletes all history records
func (h *HistoryDB) ClearHistory() (int64, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	result, err := h.db.Exec("DELETE FROM download_history")
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}
