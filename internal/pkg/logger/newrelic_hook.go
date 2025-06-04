package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// NewRelicLogHook is a logrus hook that sends logs to New Relic Logs API
type NewRelicLogHook struct {
	config      NewRelicLogConfig
	client      *http.Client
	buffer      []map[string]interface{}
	bufferMutex sync.Mutex
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

// NewNewRelicLogHook creates a new New Relic log hook
func NewNewRelicLogHook(config NewRelicLogConfig) *NewRelicLogHook {
	hook := &NewRelicLogHook{
		config: config,
		client: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
		buffer:   make([]map[string]interface{}, 0, config.BatchSize),
		stopChan: make(chan struct{}),
	}

	// Start background goroutine for periodic flushing
	hook.wg.Add(1)
	go hook.flushLoop()

	return hook
}

// Levels returns the levels this hook handles
func (hook *NewRelicLogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire sends the log entry to New Relic (buffered)
func (hook *NewRelicLogHook) Fire(entry *logrus.Entry) error {
	if !hook.config.Enabled || hook.config.LicenseKey == "" {
		return nil // Skip if not configured
	}

	// Prepare log data for New Relic
	logData := map[string]interface{}{
		"timestamp": entry.Time.UnixMilli(),
		"message":   entry.Message,
		"level":     entry.Level.String(),
		"service":   "nebengjek-users-app",
	}

	// Add all fields from the log entry
	for key, value := range entry.Data {
		logData[key] = value
	}

	// Add to buffer
	hook.bufferMutex.Lock()
	hook.buffer = append(hook.buffer, logData)
	shouldFlush := len(hook.buffer) >= hook.config.BatchSize
	hook.bufferMutex.Unlock()

	// Flush if buffer is full
	if shouldFlush {
		go hook.flush()
	}

	return nil
}

// flushLoop runs in background and flushes logs periodically
func (hook *NewRelicLogHook) flushLoop() {
	defer hook.wg.Done()

	ticker := time.NewTicker(time.Duration(hook.config.FlushPeriod) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hook.flush()
		case <-hook.stopChan:
			// Final flush before stopping
			hook.flush()
			return
		}
	}
}

// flush sends buffered logs to New Relic
func (hook *NewRelicLogHook) flush() {
	hook.bufferMutex.Lock()
	if len(hook.buffer) == 0 {
		hook.bufferMutex.Unlock()
		return
	}

	// Copy buffer and reset
	logs := make([]map[string]interface{}, len(hook.buffer))
	copy(logs, hook.buffer)
	hook.buffer = hook.buffer[:0] // Reset buffer
	hook.bufferMutex.Unlock()

	// Send to New Relic
	hook.sendToNewRelic(logs)
}

// sendToNewRelic sends logs to New Relic API
func (hook *NewRelicLogHook) sendToNewRelic(logs []map[string]interface{}) {
	if len(logs) == 0 {
		return
	}

	// Wrap in New Relic logs format
	payload := []map[string]interface{}{
		{
			"logs": logs,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("[ERROR] Failed to marshal log data: %v\n", err)
		return
	}

	req, err := http.NewRequest("POST", hook.config.Endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("[ERROR] Failed to create request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-Key", hook.config.LicenseKey)

	resp, err := hook.client.Do(req)
	if err != nil {
		fmt.Printf("[ERROR] Failed to send logs to New Relic: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 202 {
		fmt.Printf("[WARNING] New Relic API returned status: %d for %d logs\n", resp.StatusCode, len(logs))
	} else {
		fmt.Printf("[DEBUG] Successfully sent %d logs to New Relic, status: %d\n", len(logs), resp.StatusCode)
	}
}

// Close stops the hook and flushes remaining logs
func (hook *NewRelicLogHook) Close() {
	close(hook.stopChan)
	hook.wg.Wait()
}

// FlushNow immediately flushes any buffered logs
func (hook *NewRelicLogHook) FlushNow() {
	hook.flush()
}
