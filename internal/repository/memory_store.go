package repository

import (
	"context"
	"runbin/internal/model"
	"sync"
	"time"
)

type MemoryPasteStore struct {
	pastes map[string]*model.Paste
	mutex  sync.RWMutex
}

func NewMemoryPasteStore() *MemoryPasteStore {
	return &MemoryPasteStore{
		pastes: make(map[string]*model.Paste),
	}
}

func (s *MemoryPasteStore) Save(p *model.Paste) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	p.UpdatedAt = time.Now()
	s.pastes[p.ID] = p
	return nil
}

func (s *MemoryPasteStore) GetByID(id string) (*model.Paste, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	p, found := s.pastes[id]
	return p, found
}

func (s *MemoryPasteStore) DispatchExecutionTask(id string) error {
	return nil // 内存存储暂不实现队列功能
}

func (s *MemoryPasteStore) GetTask(ctx context.Context) (*model.Paste, error) {

	return nil, nil
}

func (s *MemoryPasteStore) Update(p *model.Paste) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.pastes[p.ID] = p
	return nil
}
