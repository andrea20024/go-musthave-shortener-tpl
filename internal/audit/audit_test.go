package audit

import (
	"encoding/json"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewNotifier(t *testing.T) {
	n := NewNotifier()

	if n == nil {
		t.Fatal("NewNotifier() returned nil")
	}

	// observers being nil is acceptable — append works on nil slices
	if n.observers == nil {
		t.Log("NewNotifier() created Notifier with nil observers slice (acceptable)")
	}
}

func TestAttach(t *testing.T) {
	n := NewNotifier()

	// Create a mock observer
	mock := &mockObserver{}

	n.Attach(mock)

	if len(n.observers) != 1 {
		t.Errorf("len(observers) = %d, want 1 after Attach", len(n.observers))
	}
}

func TestAttach_MultipleObservers(t *testing.T) {
	n := NewNotifier()

	mock1 := &mockObserver{}
	mock2 := &mockObserver{}
	mock3 := &mockObserver{}

	n.Attach(mock1)
	n.Attach(mock2)
	n.Attach(mock3)

	if len(n.observers) != 3 {
		t.Errorf("len(observers) = %d, want 3", len(n.observers))
	}
}

func TestNotifyAll_NoObservers(t *testing.T) {
	n := NewNotifier()

	// Should not panic with no observers
	e := Event{
		Timestamp: time.Now().Unix(),
		Action:    "test",
		UserID:    "user1",
		URL:       "https://example.com",
	}

	n.NotifyAll(e)
}

func TestNotifyAll_WithObservers(t *testing.T) {
	n := NewNotifier()

	mock := &countingObserver{}

	n.Attach(mock)

	e := Event{
		Timestamp: time.Now().Unix(),
		Action:    "shorten",
		UserID:    "user123",
		URL:       "https://example.com/very-long-url",
	}

	n.NotifyAll(e)

	// Give goroutine time to complete
	time.Sleep(10 * time.Millisecond)

	if mock.count != 1 {
		t.Errorf("observer Notify() called %d times, want 1", mock.count)
	}
}

func TestNotifyAll_EventContent(t *testing.T) {
	n := NewNotifier()

	mock := &eventCollector{}

	n.Attach(mock)

	e := Event{
		Timestamp: time.Now().Unix(),
		Action:    "follow",
		UserID:    "user456",
		URL:       "https://short.link/abc",
	}

	n.NotifyAll(e)

	// Give goroutine time to complete
	time.Sleep(10 * time.Millisecond)

	if len(mock.events) != 1 {
		t.Fatalf("observer received %d events, want 1", len(mock.events))
	}

	received := mock.events[0]
	if received.Action != "follow" {
		t.Errorf("event.Action = %q, want %q", received.Action, "follow")
	}
	if received.UserID != "user456" {
		t.Errorf("event.UserID = %q, want %q", received.UserID, "user456")
	}
	if received.URL != "https://short.link/abc" {
		t.Errorf("event.URL = %q, want %q", received.URL, "https://short.link/abc")
	}
}

func TestNotifyAll_MultipleObservers_AllReceive(t *testing.T) {
	n := NewNotifier()

	mock1 := &countingObserver{}
	mock2 := &countingObserver{}

	n.Attach(mock1)
	n.Attach(mock2)

	e := Event{
		Timestamp: time.Now().Unix(),
		Action:    "test",
		UserID:    "user",
		URL:       "url",
	}

	n.NotifyAll(e)

	time.Sleep(10 * time.Millisecond)

	if mock1.count != 1 {
		t.Errorf("observer1 received %d events, want 1", mock1.count)
	}
	if mock2.count != 1 {
		t.Errorf("observer2 received %d events, want 1", mock2.count)
	}
}

func TestFileReceiver_NewFileReceiver(t *testing.T) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "audit-test-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpName := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpName)

	fr, err := NewFileReceiver(tmpName)
	if err != nil {
		t.Fatalf("NewFileReceiver() error = %v", err)
	}
	// Close the receiver to release the file handle
	if fr != nil {
		fr.w.Close()
	}

	if fr == nil {
		t.Error("NewFileReceiver() returned nil")
	}
}

func TestFileReceiver_NewFileReceiver_NotExist(t *testing.T) {
	// Path that doesn't exist should create it
	tmpDir := t.TempDir()
	nonExistentPath := tmpDir + "/new-audit-file.json"

	fr, err := NewFileReceiver(nonExistentPath)
	if err != nil {
		t.Fatalf("NewFileReceiver() error for non-existent path = %v", err)
	}
	// Close the receiver to release the file handle before cleanup
	if fr != nil {
		fr.w.Close()
	}
	// Manually remove the file after closing
	_ = os.Remove(nonExistentPath)

	if fr == nil {
		t.Error("NewFileReceiver() returned nil")
	}
}

func TestFileReceiver_Notify(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "audit-file-test-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpName := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpName)

	fr, err := NewFileReceiver(tmpName)
	if err != nil {
		t.Fatalf("NewFileReceiver() error = %v", err)
	}
	defer fr.w.Close()

	e := Event{
		Timestamp: time.Now().Unix(),
		Action:    "shorten",
		UserID:    "user1",
		URL:       "https://example.com/test",
	}

	fr.Notify(e)

	// Read and verify file content
	data, err := os.ReadFile(tmpName)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	var received Event
	if err := json.Unmarshal(data, &received); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if received.Action != "shorten" {
		t.Errorf("action = %q, want %q", received.Action, "shorten")
	}
	if received.UserID != "user1" {
		t.Errorf("userID = %q, want %q", received.UserID, "user1")
	}
	if received.URL != "https://example.com/test" {
		t.Errorf("url = %q, want %q", received.URL, "https://example.com/test")
	}
}

func TestFileReceiver_MultipleNotifications(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "audit-multi-test-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpName := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpName)

	fr, err := NewFileReceiver(tmpName)
	if err != nil {
		t.Fatalf("NewFileReceiver() error = %v", err)
	}
	defer fr.w.Close()

	// Write multiple events
	for i := 0; i < 3; i++ {
		fr.Notify(Event{
			Timestamp: time.Now().Unix(),
			Action:    "test",
			UserID:    "user",
			URL:       "url",
		})
	}

	// Read all lines
	data, err := os.ReadFile(tmpName)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}

	if lines != 3 {
		t.Errorf("file has %d lines, want 3", lines)
	}
}

func TestEvent_JSONSerialization(t *testing.T) {
	e := Event{
		Timestamp: 1234567890,
		Action:    "shorten",
		UserID:    "user123",
		URL:       "https://example.com/long-url",
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("failed to marshal Event: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal Event: %v", err)
	}

	if result["action"] != "shorten" {
		t.Errorf("JSON action = %v, want %v", result["action"], "shorten")
	}
	if result["user_id"] != "user123" {
		t.Errorf("JSON user_id = %v, want %v", result["user_id"], "user123")
	}
}

// --- Mock observers for testing ---

type mockObserver struct{}

func (m *mockObserver) Notify(e Event) {}

type countingObserver struct {
	count int32
}

func (c *countingObserver) Notify(e Event) {
	atomic.AddInt32(&c.count, 1)
}

type eventCollector struct {
	mu     sync.Mutex
	events []Event
}

func (c *eventCollector) Notify(e Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, e)
}
