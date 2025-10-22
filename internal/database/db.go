package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

// DB wraps the SQLite database connection.
type DB struct {
	conn *sql.DB
}

// New creates and initializes a new database connection.
func New(dbPath string) (*DB, error) {
	logrus.Infof("Opening database: %s", dbPath)

	// Open SQLite database with proper pragmas
	conn, err := sql.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=ON&_journal_mode=WAL", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}

	// Initialize schema
	if err := db.InitSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Run migrations
	if err := db.RunMigrations(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	logrus.Debug("Database initialized successfully")
	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// Entry represents a feed entry in the database.
type Entry struct {
	ID        string
	EntryData []byte
	PostedAt  *sql.NullTime
	FetchedAt sql.NullTime
	CreatedAt sql.NullTime
}

// SaveEntry inserts a new entry or ignores if it already exists.
func (db *DB) SaveEntry(id string, entryJSON []byte) error {
	query := `
		INSERT OR IGNORE INTO entries (id, entry_data, fetched_at, posted_at)
		VALUES (?, ?, CURRENT_TIMESTAMP, NULL)
	`

	_, err := db.conn.Exec(query, id, entryJSON)
	if err != nil {
		return fmt.Errorf("failed to save entry: %w", err)
	}

	logrus.Debugf("Saved entry: %s", id)
	return nil
}

// GetUnpostedEntries retrieves entries that haven't been posted yet.
// If limit > 0, returns at most that many entries.
// Returns oldest entries first (by fetched_at).
func (db *DB) GetUnpostedEntries(limit int) ([]*Entry, error) {
	query := `
		SELECT id, entry_data, posted_at, fetched_at, created_at
		FROM entries
		WHERE posted_at IS NULL
		ORDER BY fetched_at ASC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query unposted entries: %w", err)
	}
	defer rows.Close()

	entries := make([]*Entry, 0)
	for rows.Next() {
		entry := &Entry{}
		err := rows.Scan(&entry.ID, &entry.EntryData, &entry.PostedAt, &entry.FetchedAt, &entry.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entries: %w", err)
	}

	return entries, nil
}

// MarkAsPosted updates an entry's posted_at timestamp to the current time.
func (db *DB) MarkAsPosted(id string) error {
	query := `UPDATE entries SET posted_at = CURRENT_TIMESTAMP WHERE id = ?`

	result, err := db.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to mark entry as posted: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("entry not found: %s", id)
	}

	logrus.Debugf("Marked entry as posted: %s", id)
	return nil
}

// GetStats returns statistics about entries in the database.
func (db *DB) GetStats() (total, posted, unposted int, err error) {
	// Get total count
	err = db.conn.QueryRow("SELECT COUNT(*) FROM entries").Scan(&total)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get posted count
	err = db.conn.QueryRow("SELECT COUNT(*) FROM entries WHERE posted_at IS NOT NULL").Scan(&posted)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get posted count: %w", err)
	}

	// Get unposted count
	err = db.conn.QueryRow("SELECT COUNT(*) FROM entries WHERE posted_at IS NULL").Scan(&unposted)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get unposted count: %w", err)
	}

	return total, posted, unposted, nil
}

// GetLastFetchTime returns the most recent fetch time as a string.
func (db *DB) GetLastFetchTime() (*string, error) {
	var fetchTime *string
	err := db.conn.QueryRow("SELECT MAX(fetched_at) FROM entries").Scan(&fetchTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get last fetch time: %w", err)
	}

	return fetchTime, nil
}

// GetLastPostTime returns the most recent post time as a string.
func (db *DB) GetLastPostTime() (*string, error) {
	var postTime *string
	err := db.conn.QueryRow("SELECT MAX(posted_at) FROM entries WHERE posted_at IS NOT NULL").Scan(&postTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get last post time: %w", err)
	}

	return postTime, nil
}
