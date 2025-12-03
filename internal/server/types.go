package server

import "encoding/json"

// Message represents a message sent over the WebSocket connection.
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// StartScanPayload is the payload for a start_scan message.
type StartScanPayload struct {
	Domain      string   `json:"domain"`
	WordlistKey string   `json:"wordlist_key,omitempty"`
	DNSServers  []string `json:"dns_servers,omitempty"`
	Concurrency int      `json:"concurrency,omitempty"`
	Adaptive    bool     `json:"adaptive,omitempty"`
	MaxQPS      int      `json:"maxQPS,omitempty"`
	EnableRetry bool     `json:"enable_retry,omitempty"`
}
