package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"subsonic/internal/scanner"
	"sync"
	"time"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	clients      map[*Client]bool
	broadcast    chan []byte
	register     chan *Client
	unregister   chan *Client
	debugNetwork bool
}

func NewHub(debugNetwork bool) *Hub {
	return &Hub{
		broadcast:    make(chan []byte),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		clients:      make(map[*Client]bool),
		debugNetwork: debugNetwork,
	}
}

func (h *Hub) runScan(payload StartScanPayload) {
	startTime := time.Now()
	wordlistChan := make(chan string, 1000)

	// The key can now be "common_speak" or "temp/some-uuid.txt"
	wordlistPath := filepath.Join("wordlists", payload.WordlistKey+".txt")
	
	// For temp files, the key already includes the .txt extension
	if filepath.Ext(payload.WordlistKey) == ".txt" {
		wordlistPath = filepath.Join("wordlists", payload.WordlistKey)
	}

	count, err := getWordlistLineCount(wordlistPath)
	if err != nil {
		log.Printf("error counting wordlist %s: %v", wordlistPath, err)
		return
	}
	totalTasks := count
	go streamWordlist(wordlistPath, wordlistChan)

	scn := scanner.NewScanner(h.debugNetwork)
	if len(payload.DNSServers) > 0 {
		scn.SetDNSServers(payload.DNSServers)
	}

	resultsChan := make(chan *scanner.ScanResult)
	statusChan := make(chan scanner.ScanStatus)

	go scn.Start(payload.Domain, wordlistChan, totalTasks, resultsChan, statusChan, payload.Concurrency, payload.Adaptive, payload.MaxQPS, payload.EnableRetry)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		const batchSize = 50
		const batchTimeout = 100 * time.Millisecond
		
		batch := make([]*scanner.ScanResult, 0, batchSize)
		ticker := time.NewTicker(batchTimeout)
		defer ticker.Stop()

		flush := func() {
			if len(batch) == 0 {
				return
			}
			payloadBytes, _ := json.Marshal(batch)
			msg := Message{Type: "scan_results", Payload: payloadBytes}
			msgBytes, _ := json.Marshal(msg)
			h.broadcast <- msgBytes
			for _, r := range batch {
				scanner.PutScanResult(r)
			}
			batch = make([]*scanner.ScanResult, 0, batchSize)
		}

		for {
			select {
			case result, ok := <-resultsChan:
				if !ok {
					flush()
					return
				}
				batch = append(batch, result)
				if len(batch) >= batchSize {
					flush()
				}
			case <-ticker.C:
				flush()
			}
		}
	}()

	var lastStatus scanner.ScanStatus
	go func() {
		defer wg.Done()
		for status := range statusChan {
			lastStatus = status
			var progress float64
			if status.Total > 0 {
				progress = float64(status.Scanned) / float64(status.Total)
			}

			message := ""
			switch status.Phase {
			case "main_scan":
				message = fmt.Sprintf("扫描中... (%d/%d) | 失败: %d | 并发: %d", status.Scanned, status.Total, status.Failed, status.Concurrency)
			case "retry_scan":
				message = fmt.Sprintf("重试失败域名... (%d/%d) | 并发: %d", status.Scanned, status.TotalRetrying, status.Concurrency)
			case "done":
				continue // Final summary is handled outside this loop
			}

			payload := map[string]interface{}{
				"status":         "scanning",
				"progress":       progress,
				"message":        message,
				"phase":          status.Phase,
				"total_retrying": status.TotalRetrying,
				"scanned":        status.Scanned,
				"total":          status.Total,
				"failed":         status.Failed,
				"totalRequests":  status.TotalRequests,
				"totalRetries":   status.TotalRetries,
			}
			payloadBytes, _ := json.Marshal(payload)
			msg := Message{Type: "scan_status", Payload: payloadBytes}
			msgBytes, _ := json.Marshal(msg)
			h.broadcast <- msgBytes
		}
	}()

	wg.Wait()
	duration := time.Since(startTime)

	failedRate := 0.0
	if totalTasks > 0 {
		failedRate = float64(lastStatus.Failed) / float64(totalTasks)
	}
	
	summary := fmt.Sprintf(
		"查询失败 %d 个 (%.2f%%)。共发送 %d 个请求 (重试 %d 次)，总耗时 %.2f 秒。",
		lastStatus.Failed,
		failedRate*100,
		lastStatus.TotalRequests,
		lastStatus.TotalRetries,
		duration.Seconds(),
	)

	finalPayload := map[string]interface{}{
		"status":        "done",
		"progress":      1.0,
		"message":       "扫描完成",
		"summary":       summary,
		"failed":        lastStatus.Failed,
		"totalRequests": lastStatus.TotalRequests,
		"totalRetries":  lastStatus.TotalRetries,
		"duration":      duration.Seconds(),
	}
	finalPayloadBytes, _ := json.Marshal(finalPayload)
	finalMsg := Message{Type: "scan_status", Payload: finalPayloadBytes}
	finalMsgBytes, _ := json.Marshal(finalMsg)
	h.broadcast <- finalMsgBytes
}

func getWordlistLineCount(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}

func streamWordlist(path string, wordlistChan chan<- string) {
	defer close(wordlistChan)
	file, err := os.Open(path)
	if err != nil {
		log.Printf("error opening wordlist for streaming: %v", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		wordlistChan <- scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		log.Printf("error reading wordlist: %v", err)
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
