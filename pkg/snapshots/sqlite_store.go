package snapshots

import (
    "context"
    "database/sql"
    "encoding/json"

    _ "modernc.org/sqlite"

    "github.com/pkg/errors"
)

// SQLiteSnapshotStore implements SnapshotStore backed by SQLite.
type SQLiteSnapshotStore struct {
    db *sql.DB
}

// NewSQLiteSnapshotStore initializes schema on the provided *sql.DB and returns a store.
func NewSQLiteSnapshotStore(db *sql.DB) (*SQLiteSnapshotStore, error) {
    s := &SQLiteSnapshotStore{db: db}
    if err := s.initSchema(); err != nil {
        return nil, errors.Wrap(err, "init sqlite schema")
    }
    return s, nil
}

// OpenSQLite opens (or creates) a SQLite database file and returns the store.
func OpenSQLite(path string) (*SQLiteSnapshotStore, error) {
    db, err := sql.Open("sqlite", path)
    if err != nil {
        return nil, errors.Wrap(err, "open sqlite db")
    }
    return NewSQLiteSnapshotStore(db)
}

func (s *SQLiteSnapshotStore) initSchema() error {
    _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS snapshots (
    conversation_id TEXT NOT NULL,
    entity_id       TEXT NOT NULL,
    kind            TEXT NOT NULL,
    version         INTEGER NOT NULL,
    status          TEXT,
    started_at      INTEGER,
    updated_at      INTEGER,
    snapshot_json   TEXT NOT NULL,
    flags_json      TEXT,
    PRIMARY KEY (conversation_id, entity_id)
);
CREATE INDEX IF NOT EXISTS idx_snapshots_conv_updated ON snapshots(conversation_id, updated_at);
`)
    return errors.Wrap(err, "create schema")
}

// Upsert implements SnapshotStore.Upsert.
func (s *SQLiteSnapshotStore) Upsert(ctx context.Context, snap Snapshot) error {
    b, err := MarshalSnapshot(snap)
    if err != nil {
        return errors.Wrap(err, "marshal snapshot")
    }

    var flagsJSON []byte
    if snap.Base().Flags != nil {
        flagsJSON, err = json.Marshal(snap.Base().Flags)
        if err != nil {
            return errors.Wrap(err, "marshal flags")
        }
    }

    _, err = s.db.ExecContext(ctx, `
INSERT INTO snapshots (conversation_id, entity_id, kind, version, status, started_at, updated_at, snapshot_json, flags_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(conversation_id, entity_id) DO UPDATE SET
    kind = excluded.kind,
    version = excluded.version,
    status = excluded.status,
    started_at = excluded.started_at,
    updated_at = excluded.updated_at,
    snapshot_json = excluded.snapshot_json,
    flags_json = excluded.flags_json
`,
        snap.Base().ConversationID,
        snap.Base().EntityID,
        string(snap.Base().Kind),
        snap.Base().Version,
        snap.Base().Status,
        snap.Base().StartedAt,
        snap.Base().UpdatedAt,
        string(b),
        nullableText(flagsJSON),
    )
    return errors.Wrap(err, "upsert snapshot")
}

// GetByConversation implements SnapshotStore.GetByConversation.
func (s *SQLiteSnapshotStore) GetByConversation(ctx context.Context, conversationID string, sinceVersion *int64) ([]Snapshot, error) {
    var (
        rows *sql.Rows
        err  error
    )
    if sinceVersion != nil {
        rows, err = s.db.QueryContext(ctx, `SELECT snapshot_json FROM snapshots WHERE conversation_id = ? AND version > ? ORDER BY updated_at ASC`, conversationID, *sinceVersion)
    } else {
        rows, err = s.db.QueryContext(ctx, `SELECT snapshot_json FROM snapshots WHERE conversation_id = ? ORDER BY updated_at ASC`, conversationID)
    }
    if err != nil {
        return nil, errors.Wrap(err, "query snapshots by conversation")
    }
    defer rows.Close()
    var out []Snapshot
    for rows.Next() {
        var js string
        if err := rows.Scan(&js); err != nil {
            return nil, errors.Wrap(err, "scan snapshot json")
        }
        snap, err := UnmarshalSnapshot([]byte(js))
        if err != nil {
            return nil, errors.Wrap(err, "decode snapshot json")
        }
        out = append(out, snap)
    }
    if err := rows.Err(); err != nil {
        return nil, errors.Wrap(err, "iterate snapshots")
    }
    return out, nil
}

// GetByEntity implements SnapshotStore.GetByEntity.
func (s *SQLiteSnapshotStore) GetByEntity(ctx context.Context, conversationID, entityID string) (Snapshot, bool, error) {
    row := s.db.QueryRowContext(ctx, `SELECT snapshot_json FROM snapshots WHERE conversation_id = ? AND entity_id = ?`, conversationID, entityID)
    var js string
    if err := row.Scan(&js); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, false, nil
        }
        return nil, false, errors.Wrap(err, "scan snapshot json")
    }
    snap, err := UnmarshalSnapshot([]byte(js))
    if err != nil {
        return nil, false, errors.Wrap(err, "decode snapshot json")
    }
    return snap, true, nil
}

func nullableText(b []byte) any {
    if len(b) == 0 {
        return nil
    }
    return string(b)
}

var _ SnapshotStore = (*SQLiteSnapshotStore)(nil)


