package snapshots

import (
	"context"
)

// SnapshotStore persists and retrieves typed snapshots for hydration.
type SnapshotStore interface {
	// Upsert inserts or replaces the snapshot for the (conversation_id, entity_id).
	Upsert(ctx context.Context, s Snapshot) error

	// GetByConversation returns all snapshots for a conversation.
	// If sinceVersion is non-nil, returns only snapshots with Version > *sinceVersion.
	GetByConversation(ctx context.Context, conversationID string, sinceVersion *int64) ([]Snapshot, error)

	// GetByEntity returns the snapshot for a specific entity within a conversation.
	GetByEntity(ctx context.Context, conversationID, entityID string) (Snapshot, bool, error)
}
