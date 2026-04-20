// Copyright 2024 LiveKit, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	//"strings"
	"sync"
	"time"

	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/logger"
)

const (
	defaultTurnTTL = 24 * time.Hour
)

type CloudflareTurnManager struct {
	client    *http.Client
	keyID     string
	apiToken  string
	ttl       time.Duration
	cacheFile string

	cache      *CloudflareTurnCache
	cacheMutex sync.RWMutex
	stopChan   chan struct{}
}

type CloudflareTurnCache struct {
	iceServers []*livekit.ICEServer
	expireAt   time.Time // Automatically 8-byte aligned on 64-bit systems
}

type cloudflareTurnCacheFile struct {
	ICEServers []*livekit.ICEServer `json:"iceServers"`
	ExpireAt   int64                `json:"expireAt"` // Unix timestamp, 8-byte aligned
}

type CloudflareTurnResponse struct {
	ICEServers []*CloudflareICEServer `json:"iceServers"`
}

type CloudflareICEServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username"`
	Credential string   `json:"credential"`
}

func NewCloudflareTurnManager() *CloudflareTurnManager {
	keyID := os.Getenv("CF_TURN_KEY_ID")
	apiToken := os.Getenv("CF_TURN_API_TOKEN")

	if keyID == "" || apiToken == "" {
		return nil
	}

	ttl := defaultTurnTTL
	if ttlStr := os.Getenv("CF_TURN_TTL"); ttlStr != "" {
		if d, err := time.ParseDuration(ttlStr); err == nil {
			ttl = d
		}
	}

	// Set cache file path in .livekit directory
	cacheDir := ".livekit"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		logger.Warnw("Failed to create cache directory", err)
	}
	cacheFile := filepath.Join(cacheDir, "cf_turn_cache.json")

	return &CloudflareTurnManager{
		client:    &http.Client{Timeout: 10 * time.Second},
		keyID:     keyID,
		apiToken:  apiToken,
		ttl:       ttl,
		cacheFile: cacheFile,
		stopChan:  make(chan struct{}),
	}
}

func (m *CloudflareTurnManager) Start() error {
	if m == nil {
		return nil
	}

	logger.Infow("Starting Cloudflare TURN manager", "keyID", m.keyID, "ttl", m.ttl)

	// Try to load from cache first
	if m.loadCacheFromFile() {
		logger.Infow("Loaded Cloudflare ICE servers from cache", "cacheFile", m.cacheFile)
	} else {
		logger.Infow("No valid cache found, fetching fresh credentials")
		// Initial fetch
		if err := m.refresh(); err != nil {
			logger.Warnw("Failed to fetch initial Cloudflare TURN credentials", err)
		}
	}

	// Start dynamic refresh loop
	go m.refreshLoop()

	return nil
}

func (m *CloudflareTurnManager) Stop() {
	if m == nil {
		return
	}

	close(m.stopChan)
}

func (m *CloudflareTurnManager) GetICEServers() []*livekit.ICEServer {
	if m == nil {
		return nil
	}

	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()

	if m.cache == nil || time.Now().After(m.cache.expireAt) {
		return nil
	}

	return m.cache.iceServers
}

func (m *CloudflareTurnManager) refreshLoop() {
	for {
		// Calculate time until next refresh
		nextRefresh := m.calculateNextRefresh()

		// Wait for next refresh or stop signal
		select {
		case <-time.After(nextRefresh):
			if err := m.refresh(); err != nil {
				logger.Warnw("Failed to refresh Cloudflare TURN credentials", err)
				// On failure, retry in 1 minute
				time.Sleep(1 * time.Minute)
			}
		case <-m.stopChan:
			return
		}
	}
}

// Calculate time until next refresh based on cache expiration
func (m *CloudflareTurnManager) calculateNextRefresh() time.Duration {
	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()

	if m.cache == nil {
		// No cache, refresh immediately
		return 0
	}

	remaining := time.Until(m.cache.expireAt)

	// Refresh 1 hour before expiration
	refreshIn := remaining - 1*time.Hour

	if refreshIn <= 0 {
		// Already expired or about to expire, refresh now
		return 0
	}

	if refreshIn > 24*time.Hour {
		// Cap at 24 hours
		return 24 * time.Hour
	}

	return refreshIn
}

func (m *CloudflareTurnManager) refresh() error {
	logger.Infow("Refreshing Cloudflare TURN credentials...")

	url := fmt.Sprintf("https://rtc.live.cloudflare.com/v1/turn/keys/%s/credentials/generate-ice-servers", m.keyID)

	// Build request body with TTL
	reqBody := map[string]int64{"ttl": int64(m.ttl.Seconds())}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", m.apiToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch Cloudflare TURN: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != 201 {
		// Read error body for debugging
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code from Cloudflare API: %d, body: %s", resp.StatusCode, string(body))
	}

	var cfResp CloudflareTurnResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert Cloudflare ICEServers to LiveKit format (includes both STUN and TURN)
	iceServers := m.convertICEServers(cfResp.ICEServers)

	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()

	m.cache = &CloudflareTurnCache{
		iceServers: iceServers,
		expireAt:   time.Now().Add(m.ttl - 60*time.Second), // Expire 60s before actual expiration
	}

	// Save to file
	if err := m.saveCacheToFile(); err != nil {
		logger.Warnw("Failed to save cache to file", err)
	}

	logger.Infow("Successfully cached Cloudflare ICE servers", "count", len(iceServers))

	return nil
}

func (m *CloudflareTurnManager) convertICEServers(cfServers []*CloudflareICEServer) []*livekit.ICEServer {
	var iceServers []*livekit.ICEServer

	for _, server := range cfServers {
		// Keep both STUN and TURN servers
		iceServers = append(iceServers, &livekit.ICEServer{
			Urls:       server.URLs,
			Username:   server.Username,
			Credential: server.Credential,
		})
	}

	return iceServers
}

func (m *CloudflareTurnManager) saveCacheToFile() error {
	if m.cache == nil {
		return fmt.Errorf("no cache to save")
	}

	cacheFile := cloudflareTurnCacheFile{
		ICEServers: m.cache.iceServers,
		ExpireAt:   m.cache.expireAt.Unix(),
	}

	data, err := json.MarshalIndent(cacheFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.WriteFile(m.cacheFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	logger.Debugw("Saved Cloudflare TURN cache to file", "file", m.cacheFile)
	return nil
}

func (m *CloudflareTurnManager) loadCacheFromFile() bool {
	data, err := os.ReadFile(m.cacheFile)
	if err != nil {
		logger.Debugw("Failed to read cache file", "error", err)
		return false
	}

	var cacheFile cloudflareTurnCacheFile
	if err := json.Unmarshal(data, &cacheFile); err != nil {
		logger.Warnw("Failed to parse cache file", err)
		return false
	}

	// Check if cache is still valid
	expireAt := time.Unix(cacheFile.ExpireAt, 0)
	if time.Now().After(expireAt) {
		logger.Infow("Cache file expired", "expiredAt", expireAt)
		return false
	}

	// Load cache
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()

	m.cache = &CloudflareTurnCache{
		iceServers: cacheFile.ICEServers,
		expireAt:   expireAt,
	}

	logger.Infow("Successfully loaded cache from file",
		"servers", len(cacheFile.ICEServers),
		"expiresAt", expireAt)

	return true
}
