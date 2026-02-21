package chat

import "errors"

var (
	ErrRoomNotFound        = errors.New("chat room not found")
	ErrNotRoomMember       = errors.New("you are not a member of this chat")
	ErrNotRoomAdmin        = errors.New("only room admin can perform this action")
	ErrCannotChatSelf      = errors.New("cannot start chat with yourself")
	ErrMessageNotFound     = errors.New("message not found")
	ErrChatNotAvailable    = errors.New("chat is available only for Pro users")
	ErrUserNotFound        = errors.New("user not found")
	ErrUserBlocked         = errors.New("cannot send message - user is blocked")
	ErrUserBanned          = errors.New("user is banned from the platform")
	ErrInvalidImageURL     = errors.New("invalid image URL - must be a valid HTTP(S) URL")
	ErrEmployerNotVerified = errors.New("employer account is pending verification")
	ErrRoomFull            = errors.New("room member limit reached")
	ErrAlreadyMember       = errors.New("user is already a member of this room")
	ErrCastingRequired     = errors.New("casting_id is required for casting rooms")
	ErrNoAccess            = errors.New("you don't have access to this casting chat")
	ErrUploadNotReady      = errors.New("attachment upload is not committed")
	ErrInvalidRoomType     = errors.New("invalid room type")
	ErrInvalidMembersCount = errors.New("at least one member is required")
)
