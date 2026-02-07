package service

import (
	"sync"
	"time"

	"github.com/set-night/mindapp/internal/domain"
)

type ModelsCache struct {
	mu       sync.RWMutex
	models   []domain.AIModel
	cachedAt time.Time
	ttl      time.Duration
}

func NewModelsCache(ttl time.Duration) *ModelsCache {
	return &ModelsCache{ttl: ttl}
}

func (c *ModelsCache) Get() []domain.AIModel {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.models == nil || time.Since(c.cachedAt) > c.ttl {
		return nil
	}
	return c.models
}

func (c *ModelsCache) Set(models []domain.AIModel) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.models = models
	c.cachedAt = time.Now()
}
