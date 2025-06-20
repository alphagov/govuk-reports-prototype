package reports

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// CacheEntry represents a cached item with expiration
type CacheEntry struct {
	Data      interface{}
	ExpiresAt time.Time
}

// ReportCache provides caching for report data and summaries
type ReportCache struct {
	summaries map[string]*CacheEntry
	reports   map[string]*CacheEntry
	stats     CacheStats
	mu        sync.RWMutex
}

// CacheStats provides statistics about cache usage
type CacheStats struct {
	SummaryHits   int64 `json:"summary_hits"`
	SummaryMisses int64 `json:"summary_misses"`
	ReportHits    int64 `json:"report_hits"`
	ReportMisses  int64 `json:"report_misses"`
	TotalEntries  int   `json:"total_entries"`
	LastCleanup   time.Time `json:"last_cleanup"`
}

// NewReportCache creates a new report cache
func NewReportCache() *ReportCache {
	cache := &ReportCache{
		summaries: make(map[string]*CacheEntry),
		reports:   make(map[string]*CacheEntry),
	}
	
	// Start background cleanup routine
	go cache.cleanupRoutine()
	
	return cache
}

// GetSummary retrieves cached summary data
func (c *ReportCache) GetSummary(reportID string, params ReportParams) []Summary {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.generateKey(reportID, "summary", params)
	entry, exists := c.summaries[key]
	
	if !exists || time.Now().After(entry.ExpiresAt) {
		c.stats.SummaryMisses++
		return nil
	}

	c.stats.SummaryHits++
	
	if summaries, ok := entry.Data.([]Summary); ok {
		return summaries
	}
	
	return nil
}

// SetSummary caches summary data
func (c *ReportCache) SetSummary(reportID string, params ReportParams, summaries []Summary, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.generateKey(reportID, "summary", params)
	c.summaries[key] = &CacheEntry{
		Data:      summaries,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// GetReport retrieves cached report data
func (c *ReportCache) GetReport(reportID string, params ReportParams) *ReportData {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.generateKey(reportID, "report", params)
	entry, exists := c.reports[key]
	
	if !exists || time.Now().After(entry.ExpiresAt) {
		c.stats.ReportMisses++
		return nil
	}

	c.stats.ReportHits++
	
	if report, ok := entry.Data.(*ReportData); ok {
		return report
	}
	
	return nil
}

// SetReport caches report data
func (c *ReportCache) SetReport(reportID string, params ReportParams, report *ReportData, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.generateKey(reportID, "report", params)
	c.reports[key] = &CacheEntry{
		Data:      report,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// Invalidate removes cached data for a specific report
func (c *ReportCache) Invalidate(reportID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove all entries that start with the report ID
	for key := range c.summaries {
		if isKeyForReport(key, reportID) {
			delete(c.summaries, key)
		}
	}
	
	for key := range c.reports {
		if isKeyForReport(key, reportID) {
			delete(c.reports, key)
		}
	}
}

// Clear removes all cached data
func (c *ReportCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.summaries = make(map[string]*CacheEntry)
	c.reports = make(map[string]*CacheEntry)
	c.stats.LastCleanup = time.Now()
}

// GetStats returns cache statistics
func (c *ReportCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := c.stats
	stats.TotalEntries = len(c.summaries) + len(c.reports)
	return stats
}

// generateKey creates a cache key from report ID, type, and parameters
func (c *ReportCache) generateKey(reportID, dataType string, params ReportParams) string {
	// Create a deterministic key based on reportID, type, and relevant parameters
	keyData := struct {
		ReportID     string
		DataType     string
		StartTime    *time.Time
		EndTime      *time.Time
		Applications []string
		Teams        []string
		Environments []string
		Filters      map[string]interface{}
		GroupBy      []string
		SortBy       string
		SortOrder    string
		Limit        int
		Offset       int
		Format       string
	}{
		ReportID:     reportID,
		DataType:     dataType,
		StartTime:    params.StartTime,
		EndTime:      params.EndTime,
		Applications: params.Applications,
		Teams:        params.Teams,
		Environments: params.Environments,
		Filters:      params.Filters,
		GroupBy:      params.GroupBy,
		SortBy:       params.SortBy,
		SortOrder:    params.SortOrder,
		Limit:        params.Limit,
		Offset:       params.Offset,
		Format:       params.Format,
	}

	jsonData, _ := json.Marshal(keyData)
	hash := md5.Sum(jsonData)
	return fmt.Sprintf("%x", hash)
}

// isKeyForReport checks if a cache key belongs to a specific report
func isKeyForReport(key, reportID string) bool {
	// This is a simple check since our keys are hashed
	// In practice, we might want to maintain a separate index
	return true // For now, invalidate all when requested
}

// cleanupRoutine runs periodically to remove expired entries
func (c *ReportCache) cleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes expired cache entries
func (c *ReportCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	
	// Clean expired summaries
	for key, entry := range c.summaries {
		if now.After(entry.ExpiresAt) {
			delete(c.summaries, key)
		}
	}
	
	// Clean expired reports
	for key, entry := range c.reports {
		if now.After(entry.ExpiresAt) {
			delete(c.reports, key)
		}
	}

	c.stats.LastCleanup = now
}