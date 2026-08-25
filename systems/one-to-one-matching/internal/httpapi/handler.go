package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/konojunya/til/systems/one-to-one-matching/internal/matching"
)

type LikeSender interface {
	SendLike(ctx context.Context, sender, receiver matching.UserID) (matching.SendLikeResult, error)
}

type MatchLister interface {
	ListMatches(
		ctx context.Context,
		userID matching.UserID,
	) ([]matching.Pair, error)
}

type MatchingApplication interface {
	LikeSender
	MatchLister
}

type handler struct {
	application MatchingApplication
}

type getMatchesResponse struct {
	Matches []matchResponse `json:"matches"`
}

type matchResponse struct {
	UserLowID  string `json:"user_low_id"`
	UserHighID string `json:"user_high_id"`
}

type putLikeResponse struct {
	LikeCreated  bool `json:"like_created"`
	MatchCreated bool `json:"match_created"`
}

type errorResponse struct {
	Error responseError `json:"error"`
}

type responseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewHandler(application MatchingApplication) http.Handler {
	h := &handler{
		application: application,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("PUT /users/{senderID}/likes/{receiverID}", h.putLike)
	mux.HandleFunc("/users/{senderID}/likes/{receiverID}", methodNotAllowed(http.MethodPut))

	mux.HandleFunc("GET /users/{userID}/matches", h.getMatches)
	mux.HandleFunc("/users/{userID}/matches", methodNotAllowed(http.MethodGet))

	return mux
}

func (h *handler) putLike(w http.ResponseWriter, r *http.Request) {
	sender, err := matching.NewUserID(r.PathValue("senderID"))
	if err != nil {
		http.Error(w, "invalid sender ID", http.StatusUnprocessableEntity)
		return
	}

	receiver, err := matching.NewUserID(r.PathValue("receiverID"))
	if err != nil {
		http.Error(w, "invalid receiver ID", http.StatusUnprocessableEntity)
		return
	}

	result, err := h.application.SendLike(r.Context(), sender, receiver)
	if err != nil {
		if errors.Is(err, matching.ErrSelfLike) {
			writeJSON(w, http.StatusUnprocessableEntity, errorResponse{
				Error: responseError{
					Code:    "SELF_LIKE",
					Message: "sender and receiver must be different",
				},
			})
			return
		}

		if errors.Is(err, matching.ErrUserNotFound) {
			writeJSON(w, http.StatusNotFound, errorResponse{
				Error: responseError{
					Code:    "USER_NOT_FOUND",
					Message: "user not found",
				},
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Error: responseError{
				Code:    "INTERNAL_ERROR",
				Message: "internal server error",
			},
		})
		return
	}

	response := putLikeResponse{
		LikeCreated:  result.LikeCreated,
		MatchCreated: result.MatchCreated,
	}

	status := http.StatusOK
	if result.LikeCreated {
		status = http.StatusCreated
	}

	writeJSON(w, status, response)
}

func (h *handler) getMatches(w http.ResponseWriter, r *http.Request) {
	userID, err := matching.NewUserID(r.PathValue("userID"))
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, errorResponse{
			Error: responseError{
				Code:    "INVALID_USER_ID",
				Message: "invalid user ID",
			},
		})
		return
	}

	pairs, err := h.application.ListMatches(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Error: responseError{
				Code:    "INTERNAL_ERROR",
				Message: "internal server error",
			},
		})
		return
	}

	matches := make([]matchResponse, 0, len(pairs))
	for _, pair := range pairs {
		matches = append(matches, matchResponse{
			UserLowID:  pair.Low().String(),
			UserHighID: pair.High().String(),
		})
	}

	writeJSON(w, http.StatusOK, getMatchesResponse{
		Matches: matches,
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(body)
}

func methodNotAllowed(allowedMethod string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", allowedMethod)

		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Error: responseError{
				Code:    "METHOD_NOT_ALLOWED",
				Message: "method not allowed",
			},
		})
	}
}
