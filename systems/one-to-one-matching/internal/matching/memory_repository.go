package matching

import "context"

type MemoryRepository struct {
	likes   map[Like]struct{}
	matches map[Pair]struct{}
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		likes:   make(map[Like]struct{}),
		matches: make(map[Pair]struct{}),
	}
}

func (r *MemoryRepository) SaveLike(_ context.Context, like Like) (bool, error) {
	if _, exists := r.likes[like]; exists {
		return false, nil
	}

	r.likes[like] = struct{}{}

	return true, nil
}

func (r *MemoryRepository) HasLike(_ context.Context, like Like) (bool, error) {
	_, exists := r.likes[like]
	return exists, nil
}

func (r *MemoryRepository) SaveMatch(_ context.Context, pair Pair) (bool, error) {
	if _, exists := r.matches[pair]; exists {
		return false, nil
	}

	r.matches[pair] = struct{}{}

	return true, nil
}

func (r *MemoryRepository) HasMatch(_ context.Context, pair Pair) (bool, error) {
	_, exists := r.matches[pair]
	return exists, nil
}
