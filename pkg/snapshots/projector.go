package snapshots

import (
	"context"
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
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
		log.Debug().Str("component", "projector").Str("conv_id", md.RunID).Str("entity_id", entityID).Str("kind", string(KindLogEvent)).Msg("projected log snapshot")
		return []Snapshot{snap}, nil

	case *events.EventPartialCompletionStart:
		entityID := md.ID.String()
		snap := &LLMTextSnapshot{SnapshotBase: base(KindLLMText, entityID, "running"), Role: "assistant", Streaming: true}
		log.Debug().Str("component", "projector").Str("conv_id", md.RunID).Str("entity_id", entityID).Str("kind", string(KindLLMText)).Msg("projected llm.start snapshot")
		return []Snapshot{snap}, nil

	case *events.EventPartialCompletion:
		entityID := md.ID.String()
		snap := &LLMTextSnapshot{SnapshotBase: base(KindLLMText, entityID, "running"), Role: "assistant", Text: ev.Completion, Streaming: true}
		log.Debug().Str("component", "projector").Str("conv_id", md.RunID).Str("entity_id", entityID).Str("kind", string(KindLLMText)).Msg("projected llm.delta snapshot")
		return []Snapshot{snap}, nil

	case *events.EventFinal:
		entityID := md.ID.String()
		snap := &LLMTextSnapshot{SnapshotBase: base(KindLLMText, entityID, "completed"), Role: "assistant", Text: ev.Text, Streaming: false}
		log.Debug().Str("component", "projector").Str("conv_id", md.RunID).Str("entity_id", entityID).Str("kind", string(KindLLMText)).Msg("projected llm.final snapshot")
		return []Snapshot{snap}, nil

	case *events.EventInterrupt:
		entityID := md.ID.String()
		snap := &LLMTextSnapshot{SnapshotBase: base(KindLLMText, entityID, "completed"), Role: "assistant", Streaming: false}
		log.Debug().Str("component", "projector").Str("conv_id", md.RunID).Str("entity_id", entityID).Str("kind", string(KindLLMText)).Msg("projected llm.interrupt snapshot")
		return []Snapshot{snap}, nil

	case *events.EventToolCall:
		entityID := ev.ToolCall.ID
		snap := &ToolCallSnapshot{SnapshotBase: base(KindToolCall, entityID, "running"), Name: ev.ToolCall.Name, Input: []byte(ev.ToolCall.Input), Exec: false}
		log.Debug().Str("component", "projector").Str("conv_id", md.RunID).Str("entity_id", entityID).Str("kind", string(KindToolCall)).Msg("projected tool.start snapshot")
		return []Snapshot{snap}, nil

	case *events.EventToolCallExecute:
		entityID := ev.ToolCall.ID
		snap := &ToolCallSnapshot{SnapshotBase: base(KindToolCall, entityID, "running"), Name: ev.ToolCall.Name, Input: []byte(ev.ToolCall.Input), Exec: true}
		log.Debug().Str("component", "projector").Str("conv_id", md.RunID).Str("entity_id", entityID).Str("kind", string(KindToolCall)).Msg("projected tool.delta snapshot")
		return []Snapshot{snap}, nil

	case *events.EventToolResult:
		entityID := ev.ToolResult.ID
		snap := &ToolResultSnapshot{SnapshotBase: base(KindToolResult, entityID, "completed"), Name: "",
			Result: []byte(ev.ToolResult.Result)}
		log.Debug().Str("component", "projector").Str("conv_id", md.RunID).Str("entity_id", entityID).Str("kind", string(KindToolResult)).Msg("projected tool.result snapshot")
		return []Snapshot{snap}, nil

	case *events.EventToolCallExecutionResult:
		entityID := ev.ToolResult.ID
		snap := &ToolResultSnapshot{SnapshotBase: base(KindToolResult, entityID, "completed"), Name: "",
			Result: []byte(ev.ToolResult.Result)}
		log.Debug().Str("component", "projector").Str("conv_id", md.RunID).Str("entity_id", entityID).Str("kind", string(KindToolResult)).Msg("projected tool.execresult snapshot")
		return []Snapshot{snap}, nil

	case *events.EventAgentModeSwitch:
		entityID := "agentmode-" + md.TurnID + "-" + md.ID.String()
		snap := &AgentModeSnapshot{SnapshotBase: base(KindAgentMode, entityID, "completed"), Title: ev.Message, Data: ev.Data}
		log.Debug().Str("component", "projector").Str("conv_id", md.RunID).Str("entity_id", entityID).Str("kind", string(KindAgentMode)).Msg("projected agent.mode snapshot")
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
	// Prefer conv_id provided via context, falling back to event metadata's RunID
	convIDFromCtx, okConv := ConversationIDFromContext(ctx)
	for _, s := range snaps {
		if s == nil {
			continue
		}
		if base := s.Base(); base != nil {
			if okConv && convIDFromCtx != "" {
				// Override to ensure snapshots are keyed by conv_id (not run_id)
				base.ConversationID = convIDFromCtx
			}
		}
		if err := store.Upsert(ctx, s); err != nil {
			return errors.Wrap(err, "upsert snapshot")
		}
		base := s.Base()
		if base != nil {
			log.Debug().
				Str("component", "snapshot_store").
				Str("conv_id", base.ConversationID).
				Str("entity_id", base.EntityID).
				Str("kind", string(base.Kind)).
				Int64("version", base.Version).
				Msg("snapshot upserted")
		}
	}
	return nil
}
