package experience

// CreateRequest represents add experience request
type CreateRequest struct {
	Title       string `json:"title" validate:"required,min=2,max=200"`
	Company     string `json:"company" validate:"omitempty,max=200"`
	Role        string `json:"role" validate:"omitempty,max=100"`
	Year        int    `json:"year" validate:"omitempty,gte=1900,lte=2100"`
	Description string `json:"description" validate:"omitempty,max=2000"`
}
