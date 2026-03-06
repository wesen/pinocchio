package webchat

// ChatServiceConfig currently mirrors ConversationServiceConfig while call sites
// converge on the tighter chat-focused surface.
type ChatServiceConfig = ConversationServiceConfig

// ChatService is a compatibility alias for the concrete conversation service.
// The old wrapper layer no longer adds behavior of its own.
type ChatService = ConversationService

func NewChatService(cfg ChatServiceConfig) (*ConversationService, error) {
	return NewConversationService(cfg)
}

func NewChatServiceFromConversation(svc *ConversationService) *ConversationService {
	return svc
}
