package middleware

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/pkg/response"
)

}

	}
}

// RequireResponseLimit enforces monthly response limits.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserID(r.Context())
		if userID == uuid.Nil {
			response.Unauthorized(w, "unauthorized")
			return
		}

		if err != nil {
			response.InternalError(w)
			return
		}

			return
		}

		next.ServeHTTP(w, r)
	})
}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserID(r.Context())
		if userID == uuid.Nil {
			response.Unauthorized(w, "unauthorized")
			return
		}

			return
		}

		next.ServeHTTP(w, r)
	})
}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserID(r.Context())
		if userID == uuid.Nil {
			response.Unauthorized(w, "unauthorized")
			return
		}

		if err != nil {
			response.InternalError(w)
			return
		}
		next.ServeHTTP(w, r)
}

		return
	}
	response.InternalError(w)
}
