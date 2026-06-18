package storage

import (
	"context"
	"sync"
)

type MemoryStore struct {
	mu      sync.RWMutex
	records map[string]ReviewRecord
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{records: map[string]ReviewRecord{}}
}

func (s *MemoryStore) SaveReview(ctx context.Context, record ReviewRecord) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[record.ReviewID] = record
	return nil
}

func (s *MemoryStore) GetReview(ctx context.Context, reviewID string) (ReviewRecord, error) {
	select {
	case <-ctx.Done():
		return ReviewRecord{}, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.records[reviewID]
	if !ok {
		return ReviewRecord{}, ErrNotFound
	}
	return record, nil
}
