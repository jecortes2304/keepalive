package recording

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func NewStore(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		err := db.Close()
		if err != nil {
			return nil, err
		}
		return nil, err
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS recordings (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			name        TEXT UNIQUE NOT NULL,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			duration_ms INTEGER NOT NULL
		);

		CREATE TABLE IF NOT EXISTS movement_points (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			recording_id INTEGER NOT NULL REFERENCES recordings(id) ON DELETE CASCADE,
			seq          INTEGER NOT NULL,
			x            INTEGER NOT NULL,
			y            INTEGER NOT NULL,
			delay_ms     INTEGER NOT NULL,
			UNIQUE(recording_id, seq)
		);
	`)
	if err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}
	return nil
}

func (s *Store) Save(rec *Recording) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.Exec(
		"INSERT INTO recordings (name, duration_ms) VALUES (?, ?)",
		rec.Name, rec.DurationMs,
	)
	if err != nil {
		return fmt.Errorf("inserting recording: %w", err)
	}

	id, _ := result.LastInsertId()
	rec.ID = id

	stmt, err := tx.Prepare("INSERT INTO movement_points (recording_id, seq, x, y, delay_ms) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range rec.Points {
		if _, err := stmt.Exec(id, p.Seq, p.X, p.Y, p.DelayMs); err != nil {
			return fmt.Errorf("inserting point %d: %w", p.Seq, err)
		}
	}

	return tx.Commit()
}

func (s *Store) List() ([]Recording, error) {
	rows, err := s.db.Query("SELECT id, name, created_at, duration_ms FROM recordings ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recordings []Recording
	for rows.Next() {
		var r Recording
		if err := rows.Scan(&r.ID, &r.Name, &r.CreatedAt, &r.DurationMs); err != nil {
			return nil, err
		}
		recordings = append(recordings, r)
	}
	return recordings, rows.Err()
}

func (s *Store) Get(name string) (*Recording, error) {
	var r Recording
	err := s.db.QueryRow(
		"SELECT id, name, created_at, duration_ms FROM recordings WHERE name = ?", name,
	).Scan(&r.ID, &r.Name, &r.CreatedAt, &r.DurationMs)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("recording %q not found", name)
	}
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(
		"SELECT seq, x, y, delay_ms FROM movement_points WHERE recording_id = ? ORDER BY seq", r.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p MovementPoint
		if err := rows.Scan(&p.Seq, &p.X, &p.Y, &p.DelayMs); err != nil {
			return nil, err
		}
		r.Points = append(r.Points, p)
	}

	return &r, rows.Err()
}

func (s *Store) Delete(name string) error {
	result, err := s.db.Exec("DELETE FROM recordings WHERE name = ?", name)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("recording %q not found", name)
	}
	return nil
}

func (s *Store) Exists(name string) bool {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM recordings WHERE name = ?", name).Scan(&count)
	return count > 0
}

func (s *Store) Rename(oldName, newName string) error {
	if s.Exists(newName) {
		return fmt.Errorf("recording %q already exists", newName)
	}
	result, err := s.db.Exec("UPDATE recordings SET name = ? WHERE name = ?", newName, oldName)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("recording %q not found", oldName)
	}
	return nil
}

func FormatDuration(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}
