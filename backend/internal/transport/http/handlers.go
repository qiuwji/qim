package http

type Handlers struct {
	Conv   *ConversationHandler
	User   *UserHandler
	Msg    *MessageHandler
	Friend *FriendHandler
	File   *FileHandler
}
