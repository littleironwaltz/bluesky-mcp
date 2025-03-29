package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	cache := New()
	
	if cache.items == nil {
		t.Error("Expected items map to be initialized")
	}
	
	if cache.fallbackItems == nil {
		t.Error("Expected fallbackItems map to be initialized")
	}
	
	if cache.options.DefaultTTL != DefaultCacheOptions.DefaultTTL {
		t.Errorf("Expected DefaultTTL to be %v, got %v", 
			DefaultCacheOptions.DefaultTTL, cache.options.DefaultTTL)
	}
	
	cache.Stop() // Clean up
}

func TestNewWithOptions(t *testing.T) {
	options := CacheOptions{
		MaxItems:         500,
		DefaultTTL:       1 * time.Minute,
		CleanupInterval:  30 * time.Second,
		AllowStaleOnFail: false,
		StaleTimeout:     5 * time.Minute,
	}
	
	cache := NewWithOptions(options)
	
	if cache.options.MaxItems != options.MaxItems {
		t.Errorf("Expected MaxItems to be %d, got %d", options.MaxItems, cache.options.MaxItems)
	}
	
	if cache.options.DefaultTTL != options.DefaultTTL {
		t.Errorf("Expected DefaultTTL to be %v, got %v", options.DefaultTTL, cache.options.DefaultTTL)
	}
	
	if cache.options.AllowStaleOnFail != options.AllowStaleOnFail {
		t.Errorf("Expected AllowStaleOnFail to be %v, got %v", 
			options.AllowStaleOnFail, cache.options.AllowStaleOnFail)
	}
	
	cache.Stop() // Clean up
}

func TestSetAndGet(t *testing.T) {
	cache := New()
	defer cache.Stop()
	
	// Set and retrieve a value
	cache.Set("key1", "value1", 1*time.Hour)
	
	value, found := cache.Get("key1")
	if !found {
		t.Error("Expected to find key1 in cache")
	}
	
	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}
	
	// Test with non-existent key
	_, found = cache.Get("nonexistent")
	if found {
		t.Error("Expected to not find nonexistent key")
	}
	
	// Test with expired item
	cache.Set("expired", "value", 1*time.Nanosecond)
	time.Sleep(10 * time.Millisecond) // Wait for expiration
	
	_, found = cache.Get("expired")
	if found {
		t.Error("Expected expired item to not be found")
	}
}

func TestGetWithRenewal(t *testing.T) {
	cache := New()
	defer cache.Stop()
	
	shortDuration := 100 * time.Millisecond
	
	// Set with short expiration
	cache.Set("key", "value", shortDuration)
	
	// Wait for half the expiration time
	time.Sleep(shortDuration / 2)
	
	// Get with renewal
	value, found := cache.GetWithRenewal("key", 1*time.Hour)
	if !found {
		t.Error("Expected to find key in cache")
	}
	
	if value != "value" {
		t.Errorf("Expected value, got %v", value)
	}
	
	// Wait for original expiration time
	time.Sleep(shortDuration)
	
	// Item should still be in cache due to renewal
	_, found = cache.Get("key")
	if !found {
		t.Error("Expected to find renewed key in cache")
	}
}

func TestGetWithLoader(t *testing.T) {
	cache := New()
	defer cache.Stop()
	
	// Use a loader function that returns a value
	loader := func() (interface{}, error) {
		return "loaded_value", nil
	}
	
	// Key does not exist yet
	value, err := cache.GetWithLoader("key", 1*time.Hour, loader)
	if err != nil {
		t.Errorf("GetWithLoader returned error: %v", err)
	}
	
	if value != "loaded_value" {
		t.Errorf("Expected loaded_value, got %v", value)
	}
	
	// Key should now be cached
	cachedValue, found := cache.Get("key")
	if !found {
		t.Error("Expected to find key in cache after loading")
	}
	
	if cachedValue != "loaded_value" {
		t.Errorf("Expected cached value to be loaded_value, got %v", cachedValue)
	}
	
	// Test with loader that returns an error
	errorLoader := func() (interface{}, error) {
		return nil, fmt.Errorf("load error")
	}
	
	_, err = cache.GetWithLoader("error_key", 1*time.Hour, errorLoader)
	if err == nil {
		t.Error("Expected error from errorLoader, got nil")
	}
}

func TestGetWithLoaderFallback(t *testing.T) {
	options := DefaultCacheOptions
	options.AllowStaleOnFail = true
	options.StaleTimeout = 1 * time.Hour
	
	cache := NewWithOptions(options)
	defer cache.Stop()
	
	// First, add an item to the cache
	cache.Set("key", "original_value", 1*time.Millisecond)
	
	// Wait for the item to expire
	time.Sleep(10 * time.Millisecond)
	
	// Define a loader that always fails
	failingLoader := func() (interface{}, error) {
		return nil, fmt.Errorf("intentional failure")
	}
	
	// We should get the stale value even though the item has expired
	// since we have AllowStaleOnFail enabled
	value, err := cache.GetWithLoader("key", 1*time.Hour, failingLoader)
	if err != nil {
		t.Errorf("Expected no error with stale fallback, got: %v", err)
	}
	
	if value != "original_value" {
		t.Errorf("Expected original_value from fallback, got %v", value)
	}
	
	// Verify stats were incremented
	stats := cache.GetStats()
	if stats.StaleServed < 1 {
		t.Errorf("Expected StaleServed to be at least 1, got %d", stats.StaleServed)
	}
}

func TestDelete(t *testing.T) {
	cache := New()
	defer cache.Stop()
	
	// Add items to cache
	cache.Set("key1", "value1", 1*time.Hour)
	cache.Set("key2", "value2", 1*time.Hour)
	
	// Delete one item
	cache.Delete("key1")
	
	// Verify key1 is deleted
	_, found := cache.Get("key1")
	if found {
		t.Error("Expected key1 to be deleted from cache")
	}
	
	// Verify key2 is still there
	value, found := cache.Get("key2")
	if !found {
		t.Error("Expected to find key2 in cache")
	}
	
	if value != "value2" {
		t.Errorf("Expected value2, got %v", value)
	}
}

func TestClear(t *testing.T) {
	cache := New()
	defer cache.Stop()
	
	// Add items to cache
	cache.Set("key1", "value1", 1*time.Hour)
	cache.Set("key2", "value2", 1*time.Hour)
	
	// Clear cache
	cache.Clear()
	
	// Verify all items are deleted
	_, found1 := cache.Get("key1")
	_, found2 := cache.Get("key2")
	
	if found1 || found2 {
		t.Error("Expected all items to be cleared from cache")
	}
	
	// Verify cache size is zero
	stats := cache.GetStats()
	if stats.Size != 0 {
		t.Errorf("Expected cache size to be 0 after clear, got %d", stats.Size)
	}
}

func TestGetStats(t *testing.T) {
	cache := New()
	defer cache.Stop()
	
	// Initially stats should be zero
	stats := cache.GetStats()
	if stats.Hits != 0 || stats.Misses != 0 || stats.Size != 0 {
		t.Errorf("Expected initial stats to be zero, got Hits=%d, Misses=%d, Size=%d",
			stats.Hits, stats.Misses, stats.Size)
	}
	
	// Add items and perform operations to generate stats
	cache.Set("key1", "value1", 1*time.Hour)
	cache.Set("key2", "value2", 1*time.Hour)
	
	// Generate cache hits
	cache.Get("key1")
	cache.Get("key1")
	cache.Get("key2")
	
	// Generate cache miss
	cache.Get("nonexistent")
	
	// Check updated stats
	stats = cache.GetStats()
	if stats.Hits != 3 {
		t.Errorf("Expected 3 hits, got %d", stats.Hits)
	}
	
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}
	
	if stats.Size != 2 {
		t.Errorf("Expected size of 2, got %d", stats.Size)
	}
}

func TestEviction(t *testing.T) {
	// Create cache with small max items
	options := DefaultCacheOptions
	options.MaxItems = 2
	
	cache := NewWithOptions(options)
	defer cache.Stop()
	
	// Add an item and make sure it's least recently used
	cache.Set("key1", "value1", 1*time.Hour)
	// Access key1 to update its last accessed time
	cache.Get("key1")
	
	// Allow a little time to pass to ensure different timestamps
	time.Sleep(5 * time.Millisecond)
	
	// Add a second item (now we're at capacity but no eviction yet)
	cache.Set("key2", "value2", 1*time.Hour)
	
	// Make key2 the most recently used
	cache.Get("key2")
	
	// Allow a little time to pass
	time.Sleep(5 * time.Millisecond)
	
	// Key1 should now be the least recently used, add a third item to trigger eviction
	cache.Set("key3", "value3", 1*time.Hour)
	
	// Verify our items
	_, found1 := cache.Get("key1")
	_, found2 := cache.Get("key2")
	_, found3 := cache.Get("key3")
	
	// Key1 should be evicted (was least recently used)
	if found1 {
		t.Error("Expected key1 to be evicted from cache")
	}
	
	// Key2 should still be there (was most recently used before key3)
	if !found2 {
		t.Error("Expected key2 to still be in cache")
	}
	
	// Key3 should be there (most recently added)
	if !found3 {
		t.Error("Expected key3 to be in cache")
	}
	
	// Check eviction stats
	stats := cache.GetStats()
	if stats.Evictions < 1 {
		t.Errorf("Expected at least 1 eviction, got %d", stats.Evictions)
	}
}

func TestPersistence(t *testing.T) {
	// Create temporary directory for cache persistence
	tmpDir := t.TempDir()
	
	// Create cache with persistence enabled
	options := DefaultCacheOptions
	options.PersistOptions.Enabled = true
	options.PersistOptions.Directory = tmpDir
	options.PersistOptions.Filename = "test_cache.json"
	options.PersistOptions.SaveInterval = 100 * time.Millisecond
	
	cache := NewWithOptions(options)
	
	// Add some items to cache
	cache.Set("key1", "value1", 1*time.Hour)
	cache.Set("key2", "value2", 1*time.Hour)
	
	// Wait for persistence to occur
	time.Sleep(200 * time.Millisecond)
	
	// Stop the cache
	cache.Stop()
	
	// Verify the persistence file exists
	cachePath := filepath.Join(tmpDir, "test_cache.json")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Errorf("Expected cache file to exist at %s", cachePath)
	}
	
	// Create a new cache with the same persistence configuration
	// which should load from the file
	newCache := NewWithOptions(options)
	defer newCache.Stop()
	
	// Check if the items were loaded
	value1, found1 := newCache.Get("key1")
	value2, found2 := newCache.Get("key2")
	
	if !found1 || !found2 {
		t.Error("Expected items to be loaded from persisted cache")
	}
	
	if value1 != "value1" || value2 != "value2" {
		t.Errorf("Expected values to match: got value1=%v, value2=%v", value1, value2)
	}
}

func TestCleanup(t *testing.T) {
	// Create cache with short cleanup interval
	options := DefaultCacheOptions
	options.CleanupInterval = 100 * time.Millisecond
	
	cache := NewWithOptions(options)
	defer cache.Stop()
	
	// Add some items with short expiration
	cache.Set("short1", "value1", 50*time.Millisecond)
	cache.Set("short2", "value2", 50*time.Millisecond)
	cache.Set("long", "value3", 1*time.Hour)
	
	// Wait for expiration and cleanup
	time.Sleep(200 * time.Millisecond)
	
	// Check if expired items were cleaned up
	_, foundShort1 := cache.Get("short1")
	_, foundShort2 := cache.Get("short2")
	_, foundLong := cache.Get("long")
	
	if foundShort1 || foundShort2 {
		t.Error("Expected short-lived items to be cleaned up")
	}
	
	if !foundLong {
		t.Error("Expected long-lived item to still be in cache")
	}
}