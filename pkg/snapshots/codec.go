package snapshots

import (
    "encoding/json"
    "fmt"
)

// MarshalSnapshot encodes a typed snapshot as JSON.
func MarshalSnapshot(s Snapshot) ([]byte, error) {
    return json.Marshal(s)
}

// UnmarshalSnapshot decodes a JSON blob into the appropriate typed snapshot
// by inspecting the `kind` field.
func UnmarshalSnapshot(b []byte) (Snapshot, error) {
    // Peek kind
    var probe struct {
        Kind SnapshotKind `json:"kind"`
    }
    if err := json.Unmarshal(b, &probe); err != nil {
        return nil, fmt.Errorf("decode snapshot kind: %w", err)
    }
    switch probe.Kind {
    case KindLLMText:
        var out LLMTextSnapshot
        if err := json.Unmarshal(b, &out); err != nil {
            return nil, fmt.Errorf("decode llm_text: %w", err)
        }
        return &out, nil
    case KindToolCall:
        var out ToolCallSnapshot
        if err := json.Unmarshal(b, &out); err != nil {
            return nil, fmt.Errorf("decode tool_call: %w", err)
        }
        return &out, nil
    case KindToolResult:
        var out ToolResultSnapshot
        if err := json.Unmarshal(b, &out); err != nil {
            return nil, fmt.Errorf("decode tool_result: %w", err)
        }
        return &out, nil
    case KindTeamAnalysis:
        var out TeamAnalysisSnapshot
        if err := json.Unmarshal(b, &out); err != nil {
            return nil, fmt.Errorf("decode team_analysis: %w", err)
        }
        return &out, nil
    case KindAgentMode:
        var out AgentModeSnapshot
        if err := json.Unmarshal(b, &out); err != nil {
            return nil, fmt.Errorf("decode agent_mode: %w", err)
        }
        return &out, nil
    case KindLogEvent:
        var out LogEventSnapshot
        if err := json.Unmarshal(b, &out); err != nil {
            return nil, fmt.Errorf("decode log_event: %w", err)
        }
        return &out, nil
    default:
        return nil, fmt.Errorf("unknown snapshot kind: %s", probe.Kind)
    }
}


