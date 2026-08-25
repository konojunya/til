package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/konojunya/til/systems/one-to-one-matching/internal/matching"
)

const matchCreatedEventType = "match.created"

type EventIDGenerator func() string

type MatchingService struct {
	transactor *Transactor
	newEventID EventIDGenerator
}

type matchCreatedPayload struct {
	UserLowID  string `json:"user_low_id"`
	UserHighID string `json:"user_high_id"`
}

func NewMatchingService(transactor *Transactor, newEventID EventIDGenerator) *MatchingService {
	return &MatchingService{
		transactor: transactor,
		newEventID: newEventID,
	}
}

func (s *MatchingService) SendLike(ctx context.Context, sender, receiver matching.UserID) (matching.SendLikeResult, error) {
	var result matching.SendLikeResult

	err := s.transactor.WithinTransaction(ctx, func(repo *Repository) error {
		service := matching.NewService(repo)
		sendLikeResult, err := service.SendLike(ctx, sender, receiver)
		if err != nil {
			return fmt.Errorf("send like: %w", err)
		}

		result = sendLikeResult

		if !sendLikeResult.MatchCreated {
			return nil
		}

		pair, err := matching.NewPair(sender, receiver)
		if err != nil {
			return fmt.Errorf("create pair: %w", err)
		}

		payload, err := json.Marshal(matchCreatedPayload{
			UserLowID:  pair.Low().String(),
			UserHighID: pair.High().String(),
		})
		if err != nil {
			return fmt.Errorf("marshal match created payload: %w", err)
		}

		if err = repo.SaveOutboxEvent(ctx, OutboxEvent{
			ID:        s.newEventID(),
			EventType: matchCreatedEventType,
			Payload:   payload,
		}); err != nil {
			return fmt.Errorf("save outbox event: %w", err)
		}

		return nil
	})
	if err != nil {
		return matching.SendLikeResult{}, err
	}

	return result, nil
}
