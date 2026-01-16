package comtrade

import (
	"sync"
	"time"
)

// Simple LRU cache for parsed ComTrade data
type cacheEntry struct {
	meta      *Metadata
	dat       *ChannelData
	timestamp time.Time
}

type datasetCache struct {
	mu      sync.RWMutex
	cache   map[string]*cacheEntry
	maxSize int
}

func NewDatasetCache(size int) *datasetCache {
	return &datasetCache{
		cache:   make(map[string]*cacheEntry),
		maxSize: size,
	}
}

func (dc *datasetCache) Clear() {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.cache = make(map[string]*cacheEntry)
}

func (dc *datasetCache) Size() int {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return len(dc.cache)
}

func (dc *datasetCache) Get(id string) (*Metadata, *ChannelData, bool) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	entry, ok := dc.cache[id]
	if ok {
		// Update timestamp for LRU
		entry.timestamp = time.Now()
		return entry.meta, entry.dat, true
	}
	return nil, nil, false
}

func (dc *datasetCache) Set(id string, meta *Metadata, dat *ChannelData) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.cache[id] = &cacheEntry{
		meta:      meta,
		dat:       dat,
		timestamp: time.Now(),
	}

	// Simple eviction: if cache is full, remove oldest entry
	if len(dc.cache) > dc.maxSize {
		var oldest string
		var oldestTime time.Time
		for k, v := range dc.cache {
			if oldestTime.IsZero() || v.timestamp.Before(oldestTime) {
				oldest = k
				oldestTime = v.timestamp
			}
		}
		delete(dc.cache, oldest)
	}
}
