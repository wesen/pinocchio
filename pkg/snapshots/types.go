package snapshots

import (
    "encoding/json"
)

// SnapshotKind is the discriminator for typed snapshots.
type SnapshotKind string

const (
    KindLLMText      SnapshotKind = "llm_text"
    KindToolCall     SnapshotKind = "tool_call"
    KindToolResult   SnapshotKind = "tool_result"
    KindTeamAnalysis SnapshotKind = "team_analysis"
    KindAgentMode    SnapshotKind = "agent_mode"
    KindLogEvent     SnapshotKind = "log_event"
)

// SnapshotBase contains fields common to all snapshots.
type SnapshotBase struct {
    ConversationID string         `json:"conversation_id"`
    EntityID       string         `json:"entity_id"`
    Kind           SnapshotKind   `json:"kind"`
    Status         string         `json:"status"`
    StartedAt      int64          `json:"started_at"`
    UpdatedAt      int64          `json:"updated_at"`
    Version        int64          `json:"version"`
    Flags          map[string]any `json:"flags,omitempty"`
}

// Snapshot is implemented by all typed snapshots.
type Snapshot interface {
    Base() *SnapshotBase
    KindValue() SnapshotKind
}

// LLMTextSnapshot represents assistant text streaming/final state.
type LLMTextSnapshot struct {
    SnapshotBase
    Role      string `json:"role"`
    Text      string `json:"text"`
    Streaming bool   `json:"streaming"`
}

func (s *LLMTextSnapshot) Base() *SnapshotBase   { return &s.SnapshotBase }
func (s *LLMTextSnapshot) KindValue() SnapshotKind { return KindLLMText }

// ToolCallSnapshot represents a tool invocation lifecycle.
type ToolCallSnapshot struct {
    SnapshotBase
    Name  string          `json:"name"`
    Input json.RawMessage `json:"input,omitempty"`
    Exec  bool            `json:"exec,omitempty"`
}

func (s *ToolCallSnapshot) Base() *SnapshotBase   { return &s.SnapshotBase }
func (s *ToolCallSnapshot) KindValue() SnapshotKind { return KindToolCall }

// ToolResultSnapshot represents a tool result.
type ToolResultSnapshot struct {
    SnapshotBase
    Name   string          `json:"name"`
    Result json.RawMessage `json:"result"`
}

func (s *ToolResultSnapshot) Base() *SnapshotBase   { return &s.SnapshotBase }
func (s *ToolResultSnapshot) KindValue() SnapshotKind { return KindToolResult }

// TeamAnalysisSnapshot represents a domain-specific analysis entity.
type TeamAnalysisSnapshot struct {
    SnapshotBase
    TeamSize int     `json:"team_size,omitempty"`
    Depth    string  `json:"depth,omitempty"`
    Progress float64 `json:"progress,omitempty"`
    Summary  string  `json:"summary,omitempty"`
}

func (s *TeamAnalysisSnapshot) Base() *SnapshotBase   { return &s.SnapshotBase }
func (s *TeamAnalysisSnapshot) KindValue() SnapshotKind { return KindTeamAnalysis }

// AgentModeSnapshot represents mode switch announcements.
type AgentModeSnapshot struct {
    SnapshotBase
    Title string         `json:"title"`
    Data  map[string]any `json:"data,omitempty"`
}

func (s *AgentModeSnapshot) Base() *SnapshotBase   { return &s.SnapshotBase }
func (s *AgentModeSnapshot) KindValue() SnapshotKind { return KindAgentMode }

// LogEventSnapshot captures log entries promoted to timeline entities.
type LogEventSnapshot struct {
    SnapshotBase
    Level   string         `json:"level"`
    Message string         `json:"message"`
    Fields  map[string]any `json:"fields,omitempty"`
}

func (s *LogEventSnapshot) Base() *SnapshotBase   { return &s.SnapshotBase }
func (s *LogEventSnapshot) KindValue() SnapshotKind { return KindLogEvent }

// Interface assertions (compile-time safety)
var (
    _ Snapshot = (*LLMTextSnapshot)(nil)
    _ Snapshot = (*ToolCallSnapshot)(nil)
    _ Snapshot = (*ToolResultSnapshot)(nil)
    _ Snapshot = (*TeamAnalysisSnapshot)(nil)
    _ Snapshot = (*AgentModeSnapshot)(nil)
    _ Snapshot = (*LogEventSnapshot)(nil)
)


