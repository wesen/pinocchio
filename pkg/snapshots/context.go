package snapshots

import "context"

// ctxKey is an unexported type to avoid key collisions in context.
type ctxKey string

const (
	ctxKeyConvID ctxKey = "snapshots.conv_id"
)

// WithConversationID attaches a conversation id to the provided context.
func WithConversationID(ctx context.Context, convID string) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return context.WithValue(ctx, ctxKeyConvID, convID)
}

// ConversationIDFromContext extracts a conversation id if present.
func ConversationIDFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	if v := ctx.Value(ctxKeyConvID); v != nil {
		if s, ok := v.(string); ok && s != "" {
			return s, true
		}
	}
	return "", false
}
