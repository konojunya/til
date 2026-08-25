//go:build integration

package postgres

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/konojunya/til/systems/one-to-one-matching/internal/httpapi"
)

type putLikeHTTPResponse struct {
	LikeCreated  bool `json:"like_created"`
	MatchCreated bool `json:"match_created"`
}

type listMatchesHTTPResponse struct {
	Matches []struct {
		UserLowID  string `json:"user_low_id"`
		UserHighID string `json:"user_high_id"`
	} `json:"matches"`
}

func requestIntegrationJSON[T any](
	t *testing.T,
	client *http.Client,
	ctx context.Context,
	method string,
	url string,
	wantStatus int,
) T {
	t.Helper()

	request, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("client.Do() error = %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != wantStatus {
		t.Errorf(
			"%s %s status = %d, want %d",
			method,
			url,
			response.StatusCode,
			wantStatus,
		)
	}

	if got := response.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}

	var body T
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	return body
}

func TestHTTPMatchingFlowPersistsAndListsOneMatch(t *testing.T) {
	ctx, conn := connectTestPostgres(t)

	const (
		aliceValue = "http-integration-alice"
		bobValue   = "http-integration-bob"
		eventID    = "http-integration-match-created"
	)

	if _, err := conn.Exec(
		ctx,
		`DELETE FROM outbox_events WHERE id = $1`,
		eventID,
	); err != nil {
		t.Fatalf("failed to clean outbox before test: %v", err)
	}

	if err := deleteMatchingServiceFixture(
		ctx,
		conn,
		aliceValue,
		bobValue,
	); err != nil {
		t.Fatalf("failed to clean matching fixture before test: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx := context.Background()

		if _, err := conn.Exec(
			cleanupCtx,
			`DELETE FROM outbox_events WHERE id = $1`,
			eventID,
		); err != nil {
			t.Errorf("failed to clean outbox after test: %v", err)
		}

		if err := deleteMatchingServiceFixture(
			cleanupCtx,
			conn,
			aliceValue,
			bobValue,
		); err != nil {
			t.Errorf("failed to clean matching fixture after test: %v", err)
		}
	})

	if _, err := conn.Exec(
		ctx,
		`INSERT INTO users (id) VALUES ($1), ($2)`,
		aliceValue,
		bobValue,
	); err != nil {
		t.Fatalf("failed to insert users: %v", err)
	}

	eventIDCalls := 0
	service := NewMatchingService(
		NewTransactor(conn),
		func() string {
			eventIDCalls++
			return eventID
		},
	)

	server := httptest.NewServer(httpapi.NewHandler(service))
	t.Cleanup(server.Close)

	client := server.Client()
	aliceLikesBobURL := server.URL + "/users/" + aliceValue + "/likes/" + bobValue
	bobLikesAliceURL := server.URL + "/users/" + bobValue + "/likes/" + aliceValue
	aliceMatchesURL := server.URL + "/users/" + aliceValue + "/matches"

	oneWay := requestIntegrationJSON[putLikeHTTPResponse](
		t,
		client,
		ctx,
		http.MethodPut,
		aliceLikesBobURL,
		http.StatusCreated,
	)
	if !oneWay.LikeCreated || oneWay.MatchCreated {
		t.Errorf("one-way PUT response = %+v, want true/false", oneWay)
	}

	mutual := requestIntegrationJSON[putLikeHTTPResponse](
		t,
		client,
		ctx,
		http.MethodPut,
		bobLikesAliceURL,
		http.StatusCreated,
	)
	if !mutual.LikeCreated || !mutual.MatchCreated {
		t.Errorf("mutual PUT response = %+v, want true/true", mutual)
	}

	resent := requestIntegrationJSON[putLikeHTTPResponse](
		t,
		client,
		ctx,
		http.MethodPut,
		aliceLikesBobURL,
		http.StatusOK,
	)
	if resent.LikeCreated || resent.MatchCreated {
		t.Errorf("resent PUT response = %+v, want false/false", resent)
	}

	listed := requestIntegrationJSON[listMatchesHTTPResponse](
		t,
		client,
		ctx,
		http.MethodGet,
		aliceMatchesURL,
		http.StatusOK,
	)

	if len(listed.Matches) != 1 {
		t.Fatalf("GET match count = %d, want 1", len(listed.Matches))
	}

	if listed.Matches[0].UserLowID != aliceValue {
		t.Errorf(
			"GET user_low_id = %q, want %q",
			listed.Matches[0].UserLowID,
			aliceValue,
		)
	}

	if listed.Matches[0].UserHighID != bobValue {
		t.Errorf(
			"GET user_high_id = %q, want %q",
			listed.Matches[0].UserHighID,
			bobValue,
		)
	}

	if eventIDCalls != 1 {
		t.Errorf("event ID generator calls = %d, want 1", eventIDCalls)
	}

	var (
		likeCount   int
		matchCount  int
		outboxCount int
	)

	if err := conn.QueryRow(
		ctx,
		`
			SELECT
				(
					SELECT COUNT(*)
					FROM likes
					WHERE (sender_id = $1 AND receiver_id = $2)
					   OR (sender_id = $2 AND receiver_id = $1)
				),
				(
					SELECT COUNT(*)
					FROM matches
					WHERE user_low_id = $1
					  AND user_high_id = $2
				),
				(
					SELECT COUNT(*)
					FROM outbox_events
					WHERE id = $3
				)
		`,
		aliceValue,
		bobValue,
		eventID,
	).Scan(
		&likeCount,
		&matchCount,
		&outboxCount,
	); err != nil {
		t.Fatalf("failed to count persisted HTTP state: %v", err)
	}

	if likeCount != 2 {
		t.Errorf("persisted like count = %d, want 2", likeCount)
	}

	if matchCount != 1 {
		t.Errorf("persisted match count = %d, want 1", matchCount)
	}

	if outboxCount != 1 {
		t.Errorf("persisted outbox count = %d, want 1", outboxCount)
	}
}
