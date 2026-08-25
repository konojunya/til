package matching

import "context"

type Repository interface {
	SaveLike(ctx context.Context, like Like) (bool, error)
	HasLike(ctx context.Context, like Like) (bool, error)
	SaveMatch(ctx context.Context, pair Pair) (bool, error)
}

type Service struct {
	repository Repository
}

type SendLikeResult struct {
	LikeCreated  bool
	MatchCreated bool
}

func NewService(repository Repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) SendLike(ctx context.Context, sender, receiver UserID) (SendLikeResult, error) {
	like, err := NewLike(sender, receiver)
	if err != nil {
		return SendLikeResult{}, err
	}

	likeCreated, err := s.repository.SaveLike(ctx, like)
	if err != nil {
		return SendLikeResult{}, err
	}

	result := SendLikeResult{
		LikeCreated: likeCreated,
	}

	reverse, err := NewLike(receiver, sender)
	if err != nil {
		return SendLikeResult{}, err
	}

	reverseExists, err := s.repository.HasLike(ctx, reverse)
	if err != nil {
		return SendLikeResult{}, err
	}

	if !reverseExists {
		return result, nil
	}

	pair, err := NewPair(sender, receiver)
	if err != nil {
		return SendLikeResult{}, err
	}

	matchCreated, err := s.repository.SaveMatch(ctx, pair)
	if err != nil {
		return SendLikeResult{}, err
	}

	result.MatchCreated = matchCreated

	return result, nil
}
