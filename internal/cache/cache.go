// Package cache provides a simple in-memory cache with automatic expiration
package cache

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Item represents a cached item with expiration
type Item struct {
	Value      interface{} `json:"value"`
	Expiration int64       `json:"expiration"`
	LastAccess int64       `json:"last_access,omitempty"`
}

// Stats tracks cache statistics
type Stats struct {
	Hits           int64 `json:"hits"`
	Misses         int64 `json:"misses"`
	Size           int   `json:"size"`
	Evictions      int64 `json:"evictions"`
	PersistHits    int64 `json:"persist_hits"`
	PersistMisses  int64 `json:"persist_misses"`
	PersistWrites  int64 `json:"persist_writes"`
	PersistErrors  int64 `json:"persist_errors"`
	StaleServed    int64 `json:"stale_served"`
}

// PersistOptions defines how cache persistence works
type PersistOptions struct {
	Enabled       bool          `json:"enabled"`
	Directory     string        `json:"directory"`
	Filename      string        `json:"filename"`
	SaveInterval  time.Duration `json:"save_interval"`
	LoadOnStartup bool          `json:"load_on_startup"`
}

// CacheOptions contains configuration options for the cache
type CacheOptions struct {
	MaxItems         int           `json:"max_items"`
	DefaultTTL       time.Duration `json:"default_ttl"`
	CleanupInterval  time.Duration `json:"cleanup_interval"`
	AllowStaleOnFail bool          `json:"allow_stale_on_fail"`
	StaleTimeout     time.Duration `json:"stale_timeout"`
	PersistOptions   PersistOptions `json:"persist_options"`
}

// DefaultCacheOptions contains reasonable defaults
var DefaultCacheOptions = CacheOptions{
	MaxItems:         1000,
	DefaultTTL:       5 * time.Minute,
	CleanupInterval:  5 * time.Minute,
	AllowStaleOnFail: true,
	StaleTimeout:     30 * time.Minute,
	PersistOptions: PersistOptions{
		Enabled:       false,
		Directory:     "./cache",
		Filename:      "cache_data.json",
		SaveInterval:  10 * time.Minute,
		LoadOnStartup: true,
	},
}

// Cache represents an in-memory cache with optional persistence
type Cache struct {
	items         map[string]Item
	mu            sync.RWMutex
	stats         Stats
	statsMu       sync.RWMutex
	stopClean     chan bool
	options       CacheOptions
	persistMu     sync.Mutex
	stopPersist   chan bool
	fallbackItems map[string]Item // Used for stale-while-revalidate
}

// LoadFunc defines a function that can load/generate a value if not in cache
type LoadFunc func() (interface{}, error)

// New creates a new cache with default options
func New() *Cache {
	return NewWithOptions(DefaultCacheOptions)
}

// NewWithOptions creates a new cache with specified options
func NewWithOptions(options CacheOptions) *Cache {
	cache := &Cache{
		items:         make(map[string]Item),
		fallbackItems: make(map[string]Item),
		stopClean:     make(chan bool),
		stopPersist:   make(chan bool),
		options:       options,
	}

	// Start cleanup routine
	go cache.startCleanupTimer()

	// Start persistence if enabled
	if options.PersistOptions.Enabled {
		// Create directory if it doesn't exist
		if err := os.MkdirAll(options.PersistOptions.Directory, 0755); err != nil {
			// Log error but continue
			fmt.Printf("Error creating cache directory: %v\n", err)
		}

		// Load cache from disk on startup if enabled
		if options.PersistOptions.LoadOnStartup {
			if err := cache.loadFromDisk(); err != nil {
				// Log error but continue
				fmt.Printf("Error loading cache from disk: %v\n", err)
			}
		}

		// Start persistence routine
		go cache.startPersistTimer()
	}

	return cache
}

// Set adds an item to the cache with expiration
func (c *Cache) Set(key string, value interface{}, duration time.Duration) {
	// Use default TTL if duration is 0
	if duration == 0 {
		duration = c.options.DefaultTTL
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to evict items
	if c.options.MaxItems > 0 && len(c.items) >= c.options.MaxItems {
		c.evictOldest()
	}

	expiration := time.Now().Add(duration).UnixNano()
	c.items[key] = Item{
		Value:      value,
		Expiration: expiration,
		LastAccess: time.Now().UnixNano(),
	}

	// Make a copy for fallback
	if c.options.AllowStaleOnFail {
		c.fallbackItems[key] = Item{
			Value:      value,
			Expiration: time.Now().Add(c.options.StaleTimeout).UnixNano(),
			LastAccess: time.Now().UnixNano(),
		}
	}
}

// evictOldest removes the least recently accessed item
func (c *Cache) evictOldest() {
	var oldestKey string
	var oldestAccess int64 = time.Now().UnixNano()

	for k, v := range c.items {
		if v.LastAccess < oldestAccess {
			oldestAccess = v.LastAccess
			oldestKey = k
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
		c.incrementEvictions()
	}
}

// Get retrieves an item from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	item, found := c.items[key]
	c.mu.RUnlock()

	if !found {
		c.incrementMisses()
		return nil, false
	}

	// Check if item has expired
	if time.Now().UnixNano() > item.Expiration {
		c.incrementMisses()
		return nil, false
	}

	// Update last access time
	c.mu.Lock()
	if i, ok := c.items[key]; ok {
		i.LastAccess = time.Now().UnixNano()
		c.items[key] = i
	}
	c.mu.Unlock()

	c.incrementHits()
	return item.Value, true
}

// GetWithRenewal gets an item and renews its expiration
func (c *Cache) GetWithRenewal(key string, duration time.Duration) (interface{}, bool) {
	value, found := c.Get(key)
	if found && duration > 0 {
		c.Set(key, value, duration)
	}
	return value, found
}

// GetWithLoader tries to get a value from cache, and if missing, calls the loader function
func (c *Cache) GetWithLoader(key string, duration time.Duration, loader LoadFunc) (interface{}, error) {
	// Try to get from cache first
	if value, found := c.Get(key); found {
		return value, nil
	}

	// Not in cache, load it
	value, err := loader()
	if err != nil {
		// If we allow stale data and have a fallback item, use it
		if c.options.AllowStaleOnFail {
			c.mu.RLock()
			staleItem, hasStale := c.fallbackItems[key]
			c.mu.RUnlock()

			if hasStale && time.Now().UnixNano() <= staleItem.Expiration {
				c.incrementStaleServed()
				return staleItem.Value, nil
			}
		}
		return nil, err
	}

	// Store in cache
	c.Set(key, value, duration)
	return value, nil
}

// Delete removes an item from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
	delete(c.fallbackItems, key)
}

// Clear empties the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]Item)
	c.fallbackItems = make(map[string]Item)
}

// GetStats returns the current cache statistics
func (c *Cache) GetStats() Stats {
	c.statsMu.RLock()
	defer c.statsMu.RUnlock()
	
	// Add current size
	c.mu.RLock()
	stats := c.stats
	stats.Size = len(c.items)
	c.mu.RUnlock()
	
	return stats
}

// incrementHits increases the hit counter
func (c *Cache) incrementHits() {
	c.statsMu.Lock()
	c.stats.Hits++
	c.statsMu.Unlock()
}

// incrementMisses increases the miss counter
func (c *Cache) incrementMisses() {
	c.statsMu.Lock()
	c.stats.Misses++
	c.statsMu.Unlock()
}

// incrementEvictions increases the eviction counter
func (c *Cache) incrementEvictions() {
	c.statsMu.Lock()
	c.stats.Evictions++
	c.statsMu.Unlock()
}

// incrementStaleServed increases the stale served counter
func (c *Cache) incrementStaleServed() {
	c.statsMu.Lock()
	c.stats.StaleServed++
	c.statsMu.Unlock()
}

// Stop halts the background cleanup goroutine
func (c *Cache) Stop() {
	close(c.stopClean)
	if c.options.PersistOptions.Enabled {
		// Save one last time
		c.persistToDisk()
		close(c.stopPersist)
	}
}

// startCleanupTimer starts a ticker to periodically clean up expired items
func (c *Cache) startCleanupTimer() {
	ticker := time.NewTicker(c.options.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopClean:
			return
		}
	}
}

// startPersistTimer starts a ticker to periodically save the cache to disk
func (c *Cache) startPersistTimer() {
	ticker := time.NewTicker(c.options.PersistOptions.SaveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.persistToDisk()
		case <-c.stopPersist:
			return
		}
	}
}

// cleanup removes expired items from the cache
func (c *Cache) cleanup() {
	now := time.Now().UnixNano()
	c.mu.Lock()
	defer c.mu.Unlock()

	for k, v := range c.items {
		if now > v.Expiration {
			delete(c.items, k)
		}
	}

	// Also cleanup fallback items
	for k, v := range c.fallbackItems {
		if now > v.Expiration {
			delete(c.fallbackItems, k)
		}
	}
}

// persistToDisk saves the cache to disk
func (c *Cache) persistToDisk() {
	if !c.options.PersistOptions.Enabled {
		return
	}

	c.persistMu.Lock()
	defer c.persistMu.Unlock()

	// Create a snapshot of the cache
	c.mu.RLock()
	snapshot := make(map[string]Item, len(c.items))
	for k, v := range c.items {
		snapshot[k] = v
	}
	c.mu.RUnlock()

	// Create the file
	filePath := filepath.Join(c.options.PersistOptions.Directory, c.options.PersistOptions.Filename)
	file, err := os.Create(filePath)
	if err != nil {
		c.incrementPersistErrors()
		return
	}
	defer file.Close()

	// Write to the file
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(snapshot); err != nil {
		c.incrementPersistErrors()
		return
	}

	c.incrementPersistWrites()
}

// loadFromDisk loads the cache from disk
func (c *Cache) loadFromDisk() error {
	c.persistMu.Lock()
	defer c.persistMu.Unlock()

	filePath := filepath.Join(c.options.PersistOptions.Directory, c.options.PersistOptions.Filename)
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, not an error
			return nil
		}
		c.incrementPersistErrors()
		return err
	}
	defer file.Close()

	// Read from the file
	var snapshot map[string]Item
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&snapshot); err != nil {
		if err != io.EOF {
			c.incrementPersistErrors()
			return err
		}
		// Empty file, not an error
		return nil
	}

	// Update the cache
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().UnixNano()
	loadCount := 0

	for k, v := range snapshot {
		// Only load non-expired items
		if now <= v.Expiration {
			c.items[k] = v
			// Also create fallback items with extended TTL
			if c.options.AllowStaleOnFail {
				c.fallbackItems[k] = Item{
					Value:      v.Value,
					Expiration: time.Now().Add(c.options.StaleTimeout).UnixNano(),
					LastAccess: v.LastAccess,
				}
			}
			loadCount++
		}
	}

	c.incrementPersistHits()
	return nil
}

// incrementPersistHits increases the persist hits counter
func (c *Cache) incrementPersistHits() {
	c.statsMu.Lock()
	c.stats.PersistHits++
	c.statsMu.Unlock()
}

// incrementPersistMisses increases the persist misses counter
func (c *Cache) incrementPersistMisses() {
	c.statsMu.Lock()
	c.stats.PersistMisses++
	c.statsMu.Unlock()
}

// incrementPersistWrites increases the persist writes counter
func (c *Cache) incrementPersistWrites() {
	c.statsMu.Lock()
	c.stats.PersistWrites++
	c.statsMu.Unlock()
}

// incrementPersistErrors increases the persist errors counter
func (c *Cache) incrementPersistErrors() {
	c.statsMu.Lock()
	c.stats.PersistErrors++
	c.statsMu.Unlock()
}