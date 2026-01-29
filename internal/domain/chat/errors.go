package chat

import "errors"

var (
	ErrRoomNotFound     = errors.New("chat room not found")
	ErrNotRoomMember    = errors.New("you are not a member of this chat")
	ErrCannotChatSelf   = errors.New("cannot start chat with yourself")
	ErrMessageNotFound  = errors.New("message not found")
	ErrChatNotAvailable = errors.New("chat is available only for Pro users")
	ErrUserNotFound     = errors.New("user not found")
	ErrUserBlocked      = errors.New("cannot send message - user is blocked")
	ErrInvalidImageURL  = errors.New("invalid image URL - must be a valid HTTP(S) URL")
)
