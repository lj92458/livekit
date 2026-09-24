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
	"testing"
	"time"

	"github.com/livekit/protocol/livekit"
)

func TestFilterTurnServers(t *testing.T) {
	manager := &CloudflareTurnManager{}

	cfServers := []*CloudflareICEServer{
		{
			URLs: []string{"stun:stun.cloudflare.com:3478"},
		},
		{
			URLs:       []string{"turn:turn.cloudflare.com:3478?transport=udp"},
			Username:   "test",
			Credential: "test",
		},
		{
			URLs:       []string{"turns:turn.cloudflare.com:443?transport=tcp"},
			Username:   "test",
			Credential: "test",
		},
	}

	turnServers := manager.filterTurnServers(cfServers)

	if len(turnServers) != 2 {
		t.Errorf("Expected 2 TURN servers, got %d", len(turnServers))
	}

	// Verify STUN server is filtered out
	for _, server := range turnServers {
		isTurn := false
		for _, url := range server.Urls {
			if len(url) > 4 && (url[:4] == "turn") {
				isTurn = true
				break
			}
		}
		if !isTurn {
			t.Errorf("Found non-TURN server in filtered results: %v", server.Urls)
		}
	}
}

func TestCloudflareTurnManagerNil(t *testing.T) {
	// Test when environment variables are not set
	manager := NewCloudflareTurnManager()
	if manager != nil {
		t.Error("Expected nil manager when env vars are not set")
	}
}

func TestCloudflareTurnCache(t *testing.T) {
	manager := &CloudflareTurnManager{
		cache: &CloudflareTurnCache{
			iceServers: []*livekit.ICEServer{
				{
					Urls:       []string{"turn:example.com:3478"},
					Username:   "test",
					Credential: "test",
				},
			},
			expireAt: time.Now().Add(1 * time.Hour),
		},
	}

	servers := manager.GetICEServers()
	if len(servers) != 1 {
		t.Errorf("Expected 1 ICE server, got %d", len(servers))
	}
}
