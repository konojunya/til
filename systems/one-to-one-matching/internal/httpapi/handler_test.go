package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/konojunya/til/systems/one-to-one-matching/internal/matching"
)

type sendLikeFunc func(
	ctx context.Context,
	sender matching.UserID,
	receiver matching.UserID,
) (matching.SendLikeResult, error)

func (f sendLikeFunc) SendLike(
	ctx context.Context,
	sender matching.UserID,
	receiver matching.UserID,
) (matching.SendLikeResult, error) {
	return f(ctx, sender, receiver)
}

func (sendLikeFunc) ListMatches(
	context.Context,
	matching.UserID,
) ([]matching.Pair, error) {
	return nil, errors.New("ListMatches must not be called in PUT test")
}

type listMatchesFunc func(
	ctx context.Context,
	userID matching.UserID,
) ([]matching.Pair, error)

func (listMatchesFunc) SendLike(
	context.Context,
	matching.UserID,
	matching.UserID,
) (matching.SendLikeResult, error) {
	return matching.SendLikeResult{}, errors.New("SendLike must not be called in GET test")
}

func (f listMatchesFunc) ListMatches(
	ctx context.Context,
	userID matching.UserID,
) ([]matching.Pair, error) {
	return f(ctx, userID)
}

func mustPair(t *testing.T, firstValue, secondValue string) matching.Pair {
	t.Helper()

	first, err := matching.NewUserID(firstValue)
	if err != nil {
		t.Fatalf("NewUserID(%q) error = %v", firstValue, err)
	}

	second, err := matching.NewUserID(secondValue)
	if err != nil {
		t.Fatalf("NewUserID(%q) error = %v", secondValue, err)
	}

	pair, err := matching.NewPair(first, second)
	if err != nil {
		t.Fatalf("NewPair(%q, %q) error = %v", firstValue, secondValue, err)
	}

	return pair
}

func TestHandlerPutLikeCreatesOneWayLike(t *testing.T) {
	t.Parallel()

	type requestContextKey struct{}

	const requestContextValue = "put-like-request"

	var (
		calls       int
		gotSender   matching.UserID
		gotReceiver matching.UserID
	)

	sender := sendLikeFunc(func(
		ctx context.Context,
		sender matching.UserID,
		receiver matching.UserID,
	) (matching.SendLikeResult, error) {
		calls++
		gotSender = sender
		gotReceiver = receiver

		if got := ctx.Value(requestContextKey{}); got != requestContextValue {
			t.Errorf(
				"SendLike() context value = %v, want %q",
				got,
				requestContextValue,
			)
		}

		return matching.SendLikeResult{
			LikeCreated:  true,
			MatchCreated: false,
		}, nil
	})

	handler := NewHandler(sender)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPut,
		"/users/alice/likes/bob",
		nil,
	)
	request = request.WithContext(context.WithValue(
		request.Context(),
		requestContextKey{},
		requestContextValue,
	))

	handler.ServeHTTP(recorder, request)

	if calls != 1 {
		t.Fatalf("SendLike() call count = %d, want 1", calls)
	}

	if gotSender.String() != "alice" {
		t.Errorf("SendLike() sender = %q, want %q", gotSender.String(), "alice")
	}

	if gotReceiver.String() != "bob" {
		t.Errorf(
			"SendLike() receiver = %q, want %q",
			gotReceiver.String(),
			"bob",
		)
	}

	if recorder.Code != http.StatusCreated {
		t.Errorf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusCreated,
		)
	}

	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}

	var response struct {
		LikeCreated  bool `json:"like_created"`
		MatchCreated bool `json:"match_created"`
	}

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	want := struct {
		LikeCreated  bool `json:"like_created"`
		MatchCreated bool `json:"match_created"`
	}{
		LikeCreated:  true,
		MatchCreated: false,
	}

	if response != want {
		t.Errorf("response body = %+v, want %+v", response, want)
	}
}

func TestHandlerPutLikeReturnsCreatedMatchForMutualLike(t *testing.T) {
	t.Parallel()

	var calls int

	sender := sendLikeFunc(func(
		context.Context,
		matching.UserID,
		matching.UserID,
	) (matching.SendLikeResult, error) {
		calls++
		return matching.SendLikeResult{
			LikeCreated:  true,
			MatchCreated: true,
		}, nil
	})

	handler := NewHandler(sender)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPut,
		"/users/bob/likes/alice",
		nil,
	)

	handler.ServeHTTP(recorder, request)

	if calls != 1 {
		t.Fatalf("SendLike() call count = %d, want 1", calls)
	}

	if recorder.Code != http.StatusCreated {
		t.Errorf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusCreated,
		)
	}

	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}

	var response struct {
		LikeCreated  bool `json:"like_created"`
		MatchCreated bool `json:"match_created"`
	}

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if !response.LikeCreated {
		t.Error("response like_created = false, want true")
	}

	if !response.MatchCreated {
		t.Error("response match_created = false, want true")
	}
}

func TestHandlerPutLikeReturnsOKForExistingLike(t *testing.T) {
	t.Parallel()

	var calls int

	sender := sendLikeFunc(func(
		context.Context,
		matching.UserID,
		matching.UserID,
	) (matching.SendLikeResult, error) {
		calls++
		return matching.SendLikeResult{}, nil
	})

	handler := NewHandler(sender)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPut,
		"/users/alice/likes/bob",
		nil,
	)

	handler.ServeHTTP(recorder, request)

	if calls != 1 {
		t.Fatalf("SendLike() call count = %d, want 1", calls)
	}

	if recorder.Code != http.StatusOK {
		t.Errorf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusOK,
		)
	}

	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}

	var response map[string]bool
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if len(response) != 2 {
		t.Fatalf("response field count = %d, want 2", len(response))
	}

	if got, ok := response["like_created"]; !ok || got {
		t.Errorf(
			"response like_created = %t, present = %t; want false, true",
			got,
			ok,
		)
	}

	if got, ok := response["match_created"]; !ok || got {
		t.Errorf(
			"response match_created = %t, present = %t; want false, true",
			got,
			ok,
		)
	}
}

func TestHandlerPutLikeReturnsStructuredErrorForSelfLike(t *testing.T) {
	t.Parallel()

	var calls int

	sender := sendLikeFunc(func(
		context.Context,
		matching.UserID,
		matching.UserID,
	) (matching.SendLikeResult, error) {
		calls++
		return matching.SendLikeResult{}, matching.ErrSelfLike
	})

	handler := NewHandler(sender)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPut,
		"/users/alice/likes/alice",
		nil,
	)

	handler.ServeHTTP(recorder, request)

	if calls != 1 {
		t.Fatalf("SendLike() call count = %d, want 1", calls)
	}

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Errorf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusUnprocessableEntity,
		)
	}

	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}

	var response struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if response.Error.Code != "SELF_LIKE" {
		t.Errorf(
			"error code = %q, want %q",
			response.Error.Code,
			"SELF_LIKE",
		)
	}

	if response.Error.Message != "sender and receiver must be different" {
		t.Errorf(
			"error message = %q, want %q",
			response.Error.Message,
			"sender and receiver must be different",
		)
	}
}

func TestHandlerPutLikeHidesInternalErrorDetails(t *testing.T) {
	t.Parallel()

	const internalDetail = "database connection failed with secret detail"

	var calls int

	sender := sendLikeFunc(func(
		context.Context,
		matching.UserID,
		matching.UserID,
	) (matching.SendLikeResult, error) {
		calls++
		return matching.SendLikeResult{}, errors.New(internalDetail)
	})

	handler := NewHandler(sender)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPut,
		"/users/alice/likes/bob",
		nil,
	)

	handler.ServeHTTP(recorder, request)

	if calls != 1 {
		t.Fatalf("SendLike() call count = %d, want 1", calls)
	}

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusInternalServerError,
		)
	}

	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}

	body := recorder.Body.String()
	if strings.Contains(body, internalDetail) {
		t.Errorf("response body leaked internal error detail: %q", body)
	}

	var response struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if response.Error.Code != "INTERNAL_ERROR" {
		t.Errorf(
			"error code = %q, want %q",
			response.Error.Code,
			"INTERNAL_ERROR",
		)
	}

	if response.Error.Message != "internal server error" {
		t.Errorf(
			"error message = %q, want %q",
			response.Error.Message,
			"internal server error",
		)
	}
}

func TestHandlerPutLikeReturnsStructuredErrorForMethodNotAllowed(t *testing.T) {
	t.Parallel()

	sender := sendLikeFunc(func(
		context.Context,
		matching.UserID,
		matching.UserID,
	) (matching.SendLikeResult, error) {
		t.Fatal("SendLike must not be called for unsupported method")
		return matching.SendLikeResult{}, nil
	})

	handler := NewHandler(sender)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/users/alice/likes/bob",
		nil,
	)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Errorf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusMethodNotAllowed,
		)
	}

	if got := recorder.Header().Get("Allow"); got != http.MethodPut {
		t.Errorf("Allow = %q, want %q", got, http.MethodPut)
	}

	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}

	var response struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if response.Error.Code != "METHOD_NOT_ALLOWED" {
		t.Errorf(
			"error code = %q, want %q",
			response.Error.Code,
			"METHOD_NOT_ALLOWED",
		)
	}

	if response.Error.Message != "method not allowed" {
		t.Errorf(
			"error message = %q, want %q",
			response.Error.Message,
			"method not allowed",
		)
	}
}

func TestHandlerPutLikeReturnsStructuredErrorForUnknownUser(t *testing.T) {
	t.Parallel()

	const internalDetail = "insert like failed with foreign key violation"

	var calls int

	sender := sendLikeFunc(func(
		context.Context,
		matching.UserID,
		matching.UserID,
	) (matching.SendLikeResult, error) {
		calls++
		return matching.SendLikeResult{}, fmt.Errorf(
			"%s: %w",
			internalDetail,
			matching.ErrUserNotFound,
		)
	})

	handler := NewHandler(sender)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPut,
		"/users/alice/likes/unknown",
		nil,
	)

	handler.ServeHTTP(recorder, request)

	if calls != 1 {
		t.Fatalf("SendLike() call count = %d, want 1", calls)
	}

	if recorder.Code != http.StatusNotFound {
		t.Errorf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusNotFound,
		)
	}

	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}

	body := recorder.Body.String()
	if strings.Contains(body, internalDetail) {
		t.Errorf("response body leaked internal error detail: %q", body)
	}

	var response struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if response.Error.Code != "USER_NOT_FOUND" {
		t.Errorf(
			"error code = %q, want %q",
			response.Error.Code,
			"USER_NOT_FOUND",
		)
	}

	if response.Error.Message != "user not found" {
		t.Errorf(
			"error message = %q, want %q",
			response.Error.Message,
			"user not found",
		)
	}
}

func TestHandlerGetMatchesReturnsEmptyArray(t *testing.T) {
	t.Parallel()

	type requestContextKey struct{}

	const requestContextValue = "get-matches-request"

	var (
		calls     int
		gotUserID matching.UserID
	)

	lister := listMatchesFunc(func(
		ctx context.Context,
		userID matching.UserID,
	) ([]matching.Pair, error) {
		calls++
		gotUserID = userID

		if got := ctx.Value(requestContextKey{}); got != requestContextValue {
			t.Errorf(
				"ListMatches() context value = %v, want %q",
				got,
				requestContextValue,
			)
		}

		// nil sliceを返しても、HTTP responseではnullではなく[]へ正規化する。
		return nil, nil
	})

	handler := NewHandler(lister)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/users/alice/matches",
		nil,
	)
	request = request.WithContext(context.WithValue(
		request.Context(),
		requestContextKey{},
		requestContextValue,
	))

	handler.ServeHTTP(recorder, request)

	if calls != 1 {
		t.Fatalf("ListMatches() call count = %d, want 1", calls)
	}

	if gotUserID.String() != "alice" {
		t.Errorf("ListMatches() user ID = %q, want %q", gotUserID.String(), "alice")
	}

	if recorder.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}

	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}

	var response struct {
		Matches []struct {
			UserLowID  string `json:"user_low_id"`
			UserHighID string `json:"user_high_id"`
		} `json:"matches"`
	}

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if response.Matches == nil {
		t.Fatal("response matches = nil, want non-nil empty array")
	}

	if len(response.Matches) != 0 {
		t.Errorf("response match count = %d, want 0", len(response.Matches))
	}
}

func TestHandlerGetMatchesReturnsPairsInApplicationOrder(t *testing.T) {
	t.Parallel()

	wantPairs := []matching.Pair{
		mustPair(t, "alice", "bob"),
		mustPair(t, "alice", "carol"),
	}

	lister := listMatchesFunc(func(
		context.Context,
		matching.UserID,
	) ([]matching.Pair, error) {
		return wantPairs, nil
	})

	handler := NewHandler(lister)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/users/alice/matches",
		nil,
	)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Matches []struct {
			UserLowID  string `json:"user_low_id"`
			UserHighID string `json:"user_high_id"`
		} `json:"matches"`
	}

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if len(response.Matches) != len(wantPairs) {
		t.Fatalf(
			"response match count = %d, want %d",
			len(response.Matches),
			len(wantPairs),
		)
	}

	for i, pair := range response.Matches {
		got := [2]string{pair.UserLowID, pair.UserHighID}
		want := [2]string{
			wantPairs[i].Low().String(),
			wantPairs[i].High().String(),
		}

		if got != want {
			t.Errorf("response matches[%d] = %q, want %q", i, got, want)
		}
	}
}

func TestHandlerGetMatchesHidesInternalErrorDetails(t *testing.T) {
	t.Parallel()

	const internalDetail = "list matches database failure with secret detail"

	lister := listMatchesFunc(func(
		context.Context,
		matching.UserID,
	) ([]matching.Pair, error) {
		return nil, errors.New(internalDetail)
	})

	handler := NewHandler(lister)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/users/alice/matches",
		nil,
	)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusInternalServerError,
		)
	}

	body := recorder.Body.String()
	if strings.Contains(body, internalDetail) {
		t.Errorf("response body leaked internal error detail: %q", body)
	}

	var response struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if response.Error.Code != "INTERNAL_ERROR" {
		t.Errorf(
			"error code = %q, want %q",
			response.Error.Code,
			"INTERNAL_ERROR",
		)
	}

	if response.Error.Message != "internal server error" {
		t.Errorf(
			"error message = %q, want %q",
			response.Error.Message,
			"internal server error",
		)
	}
}

func TestHandlerGetMatchesReturnsStructuredErrorForMethodNotAllowed(t *testing.T) {
	t.Parallel()

	lister := listMatchesFunc(func(
		context.Context,
		matching.UserID,
	) ([]matching.Pair, error) {
		t.Fatal("ListMatches must not be called for unsupported method")
		return nil, nil
	})

	handler := NewHandler(lister)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/users/alice/matches",
		nil,
	)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Errorf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusMethodNotAllowed,
		)
	}

	if got := recorder.Header().Get("Allow"); got != http.MethodGet {
		t.Errorf("Allow = %q, want %q", got, http.MethodGet)
	}

	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}

	var response struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if response.Error.Code != "METHOD_NOT_ALLOWED" {
		t.Errorf(
			"error code = %q, want %q",
			response.Error.Code,
			"METHOD_NOT_ALLOWED",
		)
	}

	if response.Error.Message != "method not allowed" {
		t.Errorf(
			"error message = %q, want %q",
			response.Error.Message,
			"method not allowed",
		)
	}
}
