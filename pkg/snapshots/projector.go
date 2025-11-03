package snapshots

import (
    "context"
    "time"

    "github.com/go-go-golems/geppetto/pkg/events"
    "github.com/pkg/errors"
)

// TimelineProjector transforms incoming backend events into typed snapshots.
// Implementations should be stateless and idempotent per event.
type TimelineProjector interface {
    Apply(ctx context.Context, ev events.Event) ([]Snapshot, error)
}

// BasicProjector provides a minimal mapping from common event types to
// snapshot records. It can be extended to support domain-specific kinds.
type BasicProjector struct{}

func NewBasicProjector() *BasicProjector { return &BasicProjector{} }

// Apply maps known events to typed snapshots. Unknown events yield no output.
func (p *BasicProjector) Apply(ctx context.Context, e events.Event) ([]Snapshot, error) {
    if e == nil {
        return nil, nil
    }
    md := e.Metadata()
    now := time.Now().UnixMilli()

    // Helper to initialize base
    base := func(kind SnapshotKind, entityID string, status string) SnapshotBase {
        return SnapshotBase{
            ConversationID: md.RunID, // best available conversation/run correlation for MVP
            EntityID:       entityID,
            Kind:           kind,
            Status:         status,
            StartedAt:      now,
            UpdatedAt:      now,
            Version:        now,
        }
    }

    switch ev := e.(type) {
    case *events.EventLog:
        entityID := md.ID.String()
        snap := &LogEventSnapshot{SnapshotBase: base(KindLogEvent, entityID, "completed"), Level: ev.Level, Message: ev.Message, Fields: ev.Fields}
        return []Snapshot{snap}, nil

    case *events.EventPartialCompletionStart:
        entityID := md.ID.String()
        snap := &LLMTextSnapshot{SnapshotBase: base(KindLLMText, entityID, "running"), Role: "assistant", Streaming: true}
        return []Snapshot{snap}, nil

    case *events.EventPartialCompletion:
        entityID := md.ID.String()
        snap := &LLMTextSnapshot{SnapshotBase: base(KindLLMText, entityID, "running"), Role: "assistant", Text: ev.Completion, Streaming: true}
        return []Snapshot{snap}, nil

    case *events.EventFinal:
        entityID := md.ID.String()
        snap := &LLMTextSnapshot{SnapshotBase: base(KindLLMText, entityID, "completed"), Role: "assistant", Text: ev.Text, Streaming: false}
        return []Snapshot{snap}, nil

    case *events.EventInterrupt:
        entityID := md.ID.String()
        snap := &LLMTextSnapshot{SnapshotBase: base(KindLLMText, entityID, "completed"), Role: "assistant", Streaming: false}
        return []Snapshot{snap}, nil

    case *events.EventToolCall:
        entityID := ev.ToolCall.ID
        snap := &ToolCallSnapshot{SnapshotBase: base(KindToolCall, entityID, "running"), Name: ev.ToolCall.Name, Input: []byte(ev.ToolCall.Input), Exec: false}
        return []Snapshot{snap}, nil

    case *events.EventToolCallExecute:
        entityID := ev.ToolCall.ID
        snap := &ToolCallSnapshot{SnapshotBase: base(KindToolCall, entityID, "running"), Name: ev.ToolCall.Name, Input: []byte(ev.ToolCall.Input), Exec: true}
        return []Snapshot{snap}, nil

    case *events.EventToolResult:
        entityID := ev.ToolResult.ID
        snap := &ToolResultSnapshot{SnapshotBase: base(KindToolResult, entityID, "completed"), Name: "",
            Result: []byte(ev.ToolResult.Result)}
        return []Snapshot{snap}, nil

    case *events.EventToolCallExecutionResult:
        entityID := ev.ToolResult.ID
        snap := &ToolResultSnapshot{SnapshotBase: base(KindToolResult, entityID, "completed"), Name: "",
            Result: []byte(ev.ToolResult.Result)}
        return []Snapshot{snap}, nil

    case *events.EventAgentModeSwitch:
        entityID := "agentmode-" + md.TurnID + "-" + md.ID.String()
        snap := &AgentModeSnapshot{SnapshotBase: base(KindAgentMode, entityID, "completed"), Title: ev.Message, Data: ev.Data}
        return []Snapshot{snap}, nil
    }

    return nil, nil
}

// ProjectAndPersist is a convenience to call the projector and upsert results.
func ProjectAndPersist(ctx context.Context, proj TimelineProjector, store SnapshotStore, e events.Event) error {
    if proj == nil || store == nil || e == nil {
        return nil
    }
    snaps, err := proj.Apply(ctx, e)
    if err != nil {
        return errors.Wrap(err, "project event")
    }
    for _, s := range snaps {
        if s == nil {
            continue
        }
        if err := store.Upsert(ctx, s); err != nil {
            return errors.Wrap(err, "upsert snapshot")
        }
    }
    return nil
}


