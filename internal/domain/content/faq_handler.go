package content

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/mwork/mwork-api/internal/pkg/response"
)

// FAQItem represents a FAQ question/answer
type FAQItem struct {
	ID        string `db:"id" json:"id"`
	Category  string `db:"category" json:"category"`
	Question  string `db:"question" json:"question"`
	Answer    string `db:"answer" json:"answer"`
	SortOrder int    `db:"sort_order" json:"sort_order"`
}

// FAQHandler handles FAQ HTTP requests
type FAQHandler struct {
	db *sqlx.DB
}

// NewFAQHandler creates FAQ handler
func NewFAQHandler(db *sqlx.DB) *FAQHandler {
	return &FAQHandler{db: db}
}

// List handles GET /faq
// @Summary Список часто задаваемых вопросов
// @Tags FAQ
// @Produce json
// @Param category query string false "Категория"
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Failure 500 {object} response.Response
// @Router /faq [get]
func (h *FAQHandler) List(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")

	var items []FAQItem
	var err error

	if category != "" {
		err = h.db.SelectContext(r.Context(), &items, `
			SELECT id, category, question, answer, sort_order
			FROM faq_items
			WHERE is_active = true AND category = $1
			ORDER BY sort_order
		`, category)
	} else {
		err = h.db.SelectContext(r.Context(), &items, `
			SELECT id, category, question, answer, sort_order
			FROM faq_items
			WHERE is_active = true
			ORDER BY sort_order
		`)
	}

	if err != nil {
		response.InternalError(w)
		return
	}

	// Group by category
	grouped := make(map[string][]FAQItem)
	for _, item := range items {
		grouped[item.Category] = append(grouped[item.Category], item)
	}

	response.OK(w, map[string]interface{}{
		"items":   items,
		"grouped": grouped,
		"total":   len(items),
	})
}

// GetCategories handles GET /faq/categories
// @Summary Список категорий вопросов
// @Tags FAQ
// @Produce json
// @Success 200 {object} response.Response{data=[]string}
// @Failure 500 {object} response.Response
// @Router /faq/categories [get]
func (h *FAQHandler) GetCategories(w http.ResponseWriter, r *http.Request) {
	var categories []string
	err := h.db.SelectContext(r.Context(), &categories, `
		SELECT DISTINCT category FROM faq_items WHERE is_active = true ORDER BY category
	`)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, categories)
}

// Routes returns FAQ routes
func (h *FAQHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Get("/categories", h.GetCategories)

	return r
}
