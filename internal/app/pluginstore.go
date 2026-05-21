package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type PluginStore struct {
	db *sql.DB
}

func OpenPluginStore(path string) (*PluginStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	store := &PluginStore{db: db}
	if err := store.init(); err != nil {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}

func (s *PluginStore) Close() error {
	return s.db.Close()
}

func (s *PluginStore) Get(pluginID string, key string, target any) (bool, error) {
	var raw string
	err := s.db.QueryRow(
		`SELECT value_json FROM plugin_store WHERE plugin_id = ? AND key = ?`,
		pluginID,
		key,
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if target == nil {
		return true, nil
	}
	if err := json.Unmarshal([]byte(raw), target); err != nil {
		return false, fmt.Errorf("decode plugin store value: %w", err)
	}
	return true, nil
}

func (s *PluginStore) Set(pluginID string, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode plugin store value: %w", err)
	}
	_, err = s.db.Exec(
		`INSERT INTO plugin_store (plugin_id, key, value_json, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(plugin_id, key) DO UPDATE SET
		   value_json = excluded.value_json,
		   updated_at = excluded.updated_at`,
		pluginID,
		key,
		string(raw),
		time.Now().UnixMilli(),
	)
	return err
}

func (s *PluginStore) Delete(pluginID string, key string) error {
	_, err := s.db.Exec(
		`DELETE FROM plugin_store WHERE plugin_id = ? AND key = ?`,
		pluginID,
		key,
	)
	return err
}

func (s *PluginStore) List(pluginID string, prefix string, target any) error {
	rows, err := s.db.Query(
		`SELECT value_json FROM plugin_store
		 WHERE plugin_id = ? AND key LIKE ? ESCAPE '\'
		 ORDER BY key`,
		pluginID,
		escapeLikePrefix(prefix)+"%",
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	if target == nil {
		return nil
	}

	rawValues := make([]json.RawMessage, 0)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return err
		}
		rawValues = append(rawValues, json.RawMessage(raw))
	}
	if err := rows.Err(); err != nil {
		return err
	}
	raw, err := json.Marshal(rawValues)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("decode plugin store list: %w", err)
	}
	return nil
}

func (s *PluginStore) init() error {
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS plugin_store (
		plugin_id TEXT NOT NULL,
		key TEXT NOT NULL,
		value_json TEXT NOT NULL,
		updated_at INTEGER NOT NULL,
		PRIMARY KEY (plugin_id, key)
	)`)
	return err
}

func escapeLikePrefix(prefix string) string {
	out := make([]byte, 0, len(prefix))
	for i := 0; i < len(prefix); i++ {
		switch prefix[i] {
		case '\\', '%', '_':
			out = append(out, '\\')
		}
		out = append(out, prefix[i])
	}
	return string(out)
}
