// Package audit implements the observer pattern for URL shortener events.
//
// It defines the Observer interface and a Notifier that broadcasts events
// (shorten, follow) to multiple receivers. Built-in receivers include
// FileReceiver (writes events to a local file) and HTTPReceiver (POSTs
// events to a remote endpoint).
package audit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"
)

// Event represents an audit log entry produced by the URL shortener.
type Event struct {
	Timestamp int64  `json:"ts"`
	Action    string `json:"action"`
	UserID    string `json:"user_id"`
	URL       string `json:"url"`
}

// Observer is the interface that audit receivers must implement to receive
// Event notifications from the Notifier.
type Observer interface {
	Notify(Event)
}

// Notifier broadcasts audit events to all attached observers using the
// observer pattern. Notify calls are dispatched asynchronously in separate
// goroutines.
type Notifier struct {
	observers []Observer
	mu        sync.Mutex
}

// NewNotifier creates a new Notifier instance with no observers attached.
// Use Attach to register receivers.
func NewNotifier() *Notifier { return &Notifier{} }

// Attach registers an Observer to receive audit events. The observer will be
// called asynchronously for each event via its Notify method.
func (n *Notifier) Attach(o Observer) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.observers = append(n.observers, o)
}

// NotifyAll broadcasts an Event to all attached observers in separate
// goroutines.
func (n *Notifier) NotifyAll(e Event) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, o := range n.observers {
		go o.Notify(e)
	}
}

// FileReceiver writes audit events to a local file as newline-delimited JSON.
type FileReceiver struct {
	w *os.File
	m sync.Mutex
}

// NewFileReceiver creates a FileReceiver that appends events to the specified
// file path. The file is created if it does not exist.
func NewFileReceiver(path string) (*FileReceiver, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileReceiver{w: f}, nil
}

// Notify writes the event as a single line of JSON to the underlying file.
func (f *FileReceiver) Notify(e Event) {
	f.m.Lock()
	defer f.m.Unlock()
	data, _ := json.Marshal(e)
	f.w.Write(append(data, '\n'))
}

// HTTPReceiver sends audit events to a remote HTTP endpoint via POST with
// application/json content type.
type HTTPReceiver struct {
	url    string
	client *http.Client
}

// NewHTTPReceiver creates an HTTPReceiver that POSTs events to the given URL.
// The HTTP client has a 10-second timeout.
func NewHTTPReceiver(url string) *HTTPReceiver {
	return &HTTPReceiver{url: url, client: &http.Client{Timeout: 10 * time.Second}}
}

// Notify sends the event as a JSON payload to the configured URL.
func (h *HTTPReceiver) Notify(e Event) {
	data, _ := json.Marshal(e)
	resp, _ := h.client.Post(h.url, "application/json", bytes.NewReader(data))
	if resp != nil {
		resp.Body.Close()
	}
}
