package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type CloudflareTurnResponse struct {
	ICEServers []struct {
		URLs       []string `json:"urls"`
		Username   string   `json:"username"`
		Credential string   `json:"credential"`
	} `json:"iceServers"`
}

func main() {
	keyID := os.Getenv("CF_TURN_KEY_ID")
	apiToken := os.Getenv("CF_TURN_API_TOKEN")
	ttl := os.Getenv("CF_TURN_TTL")

	if keyID == "" || apiToken == "" {
		fmt.Println("❌ Error: CF_TURN_KEY_ID and CF_TURN_API_TOKEN are required")
		fmt.Println("\nPlease set them:")
		fmt.Println("  export CF_TURN_KEY_ID='your-key-id'")
		fmt.Println("  export CF_TURN_API_TOKEN='your-api-token'")
		os.Exit(1)
	}

	if ttl == "" {
		ttl = "86400"
	}

	fmt.Println("==========================================")
	fmt.Println("Testing Cloudflare TURN API")
	fmt.Println("==========================================")
	fmt.Printf("Key ID: %s\n", keyID)
	fmt.Printf("TTL: %s seconds\n", ttl)
	fmt.Println()

	url := fmt.Sprintf("https://rtc.live.cloudflare.com/v1/turn/keys/%s/credentials/generate-ice-servers?ttl=%s", keyID, ttl)
	fmt.Printf("Request URL: %s\n", url)
	fmt.Println()

	// Create request
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		fmt.Printf("❌ Failed to create request: %v\n", err)
		os.Exit(1)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiToken))
	req.Header.Set("Content-Type", "application/json")

	// Send request
	fmt.Println("Sending request to Cloudflare API...")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ Failed to send request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	fmt.Printf("Response Status: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Println()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ Failed to read response: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ API Error:\n%s\n", string(body))
		os.Exit(1)
	}

	// Parse response
	var cfResp CloudflareTurnResponse
	if err := json.Unmarshal(body, &cfResp); err != nil {
		fmt.Printf("❌ Failed to parse response: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Success!")
	fmt.Println()
	fmt.Println("ICE Servers:")
	fmt.Println("----------------------------------------")
	for i, server := range cfResp.ICEServers {
		fmt.Printf("\nServer %d:\n", i+1)
		fmt.Printf("  URLs:\n")
		for _, url := range server.URLs {
			fmt.Printf("    - %s\n", url)
		}
		fmt.Printf("  Username: %s\n", server.Username)
		fmt.Printf("  Credential: %s\n", server.Credential)
	}

	fmt.Println()
	fmt.Println("==========================================")

	// Filter TURN servers
	fmt.Println("\nFiltered TURN Servers (STUN removed):")
	fmt.Println("----------------------------------------")
	turnCount := 0
	for _, server := range cfResp.ICEServers {
		hasTurn := false
		for _, url := range server.URLs {
			if len(url) >= 4 && (url[:4] == "turn") {
				hasTurn = true
				break
			}
		}
		if hasTurn {
			turnCount++
			fmt.Printf("\nTURN Server %d:\n", turnCount)
			fmt.Printf("  URLs:\n")
			for _, url := range server.URLs {
				fmt.Printf("    - %s\n", url)
			}
			fmt.Printf("  Username: %s\n", server.Username)
			fmt.Printf("  Credential: %s\n", server.Credential)
		}
	}

	fmt.Println()
	fmt.Printf("Total: %d TURN servers (STUN servers filtered out)\n", turnCount)
}
