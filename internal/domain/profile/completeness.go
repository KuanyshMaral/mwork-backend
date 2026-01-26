package profile

// CompletenessWeight defines the weight for each profile field
type CompletenessWeight struct {
	Field  string
	Weight int
	Tip    string
}

// CompletenessWeights defines all fields and their weights for profile completeness
var CompletenessWeights = []CompletenessWeight{
	// Required fields (50% total)
	{"name", 10, "Укажите ваше имя"},
	{"city", 10, "Укажите ваш город"},
	{"age", 5, "Укажите ваш возраст"},
	{"height", 10, "Укажите ваш рост"},
	{"weight", 5, "Укажите ваш вес"},
	{"bio", 10, "Добавьте описание о себе"},

	// Measurements (15% total)
	{"shoe_size", 5, "Укажите размер обуви"},
	{"clothing_size", 5, "Укажите размер одежды"},
	{"gender", 5, "Укажите ваш пол"},

	// Optional (35% total)
	{"photos", 20, "Добавьте минимум 3 фотографии для портфолио"},
	{"languages", 5, "Укажите языки, которыми владеете"},
	{"categories", 10, "Укажите категории (модель, актер и т.д.)"},
}

// CalculateModelCompleteness calculates the profile completeness percentage
func CalculateModelCompleteness(p *ModelProfile, photoCount int) *CompletenessResponse {
	score := 0
	missing := []string{}
	tips := []string{}

	// Check each field
	for _, w := range CompletenessWeights {
		filled := false
		switch w.Field {
		case "name":
			filled = p.Name.Valid && p.Name.String != ""
		case "city":
			filled = p.City.Valid && p.City.String != ""
		case "age":
			filled = p.Age.Valid && p.Age.Int32 > 0
		case "height":
			filled = p.Height.Valid && p.Height.Float64 > 0
		case "weight":
			filled = p.Weight.Valid && p.Weight.Float64 > 0
		case "bio":
			filled = p.Bio.Valid && len(p.Bio.String) > 10
		case "shoe_size":
			filled = p.ShoeSize.Valid && p.ShoeSize.String != ""
		case "clothing_size":
			filled = p.ClothingSize.Valid && p.ClothingSize.String != ""
		case "gender":
			filled = p.Gender.Valid && p.Gender.String != ""
		case "photos":
			filled = photoCount >= 3
		case "languages":
			filled = len(p.GetLanguages()) > 0
		case "categories":
			filled = len(p.GetCategories()) > 0
		}

		if filled {
			score += w.Weight
		} else {
			missing = append(missing, w.Field)
			tips = append(tips, w.Tip)
		}
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}

	// Limit tips to top 3
	if len(tips) > 3 {
		tips = tips[:3]
	}

	return &CompletenessResponse{
		Percentage:    score,
		MissingFields: missing,
		Tips:          tips,
	}
}
