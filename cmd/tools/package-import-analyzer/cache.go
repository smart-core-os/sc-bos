package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// APICache handles persistent caching of API responses
type APICache struct {
	cacheDir  string
	apiCalls  int // track total API calls made
	cacheHits int // track cache hits
}

// newAPICache creates a new API cache instance
func newAPICache(cacheDir string) (*APICache, error) {
	if cacheDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %v", err)
		}
		cacheDir = filepath.Join(homeDir, ".package-import-analyzer-cache")
	}

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %v", err)
	}

	return &APICache{
		cacheDir:  cacheDir,
		apiCalls:  0,
		cacheHits: 0,
	}, nil
}

// getCacheKey generates a cache key for a URL
func (c *APICache) getCacheKey(url string) string {
	hash := md5.Sum([]byte(url))
	return fmt.Sprintf("%x", hash)
}

// getCacheFilePath returns the full path to a cache file
// Files are organized into subdirectories named by the first two characters of the hash
func (c *APICache) getCacheFilePath(cacheKey string) string {
	// Use first two characters as subdirectory name
	subDir := cacheKey[:2]
	dirPath := filepath.Join(c.cacheDir, subDir)

	// Create subdirectory if it doesn't exist
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		// Fall back to flat structure if subdirectory creation fails
		return filepath.Join(c.cacheDir, cacheKey+".json")
	}

	return filepath.Join(dirPath, cacheKey+".json")
}

// get retrieves data from cache if it exists and is not expired
func (c *APICache) get(url string, target any) bool {
	cacheKey := c.getCacheKey(url)
	filePath := c.getCacheFilePath(cacheKey)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return false // Cache miss
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return false // Invalid cache entry
	}

	// Check if cache entry is expired
	if time.Now().After(entry.ExpiresAt) {
		os.Remove(filePath) // Clean up expired cache
		return false
	}

	// Decode the cached data into target
	entryData, err := json.Marshal(entry.Data)
	if err != nil {
		return false
	}

	if err := json.Unmarshal(entryData, target); err != nil {
		return false
	}

	c.cacheHits++
	if *veryVerbose {
		log.Printf("Cache HIT for %s", url)
	}
	return true
}

// set stores data in cache with expiration time
func (c *APICache) set(url string, data any, ttl time.Duration) error {
	cacheKey := c.getCacheKey(url)
	filePath := c.getCacheFilePath(cacheKey)

	entry := CacheEntry{
		Data:      data,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}

	entryData, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, entryData, 0644)
}
