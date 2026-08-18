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
	"fmt"
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
// observer pattern. Events are queued in a buffered channel and processed
// by a single worker goroutine to avoid unbounded goroutine growth.
type Notifier struct {
	observers []Observer
	mu        sync.RWMutex
	events    chan Event
	quit      chan struct{}
	wg        sync.WaitGroup
	startOnce sync.Once
}

// NewNotifier creates a new Notifier instance with no observers attached..
func NewNotifier() *Notifier {
	return &Notifier{
		quit: make(chan struct{}),
	}
}

// Attach registers an Observer to receive audit events. The observer will be
// called for each event via its Notify method.
func (n *Notifier) Attach(o Observer) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.observers = append(n.observers, o)
}

// Stop gracefully shuts down the worker, waiting for all pending events to
// be processed before returning.
func (n *Notifier) Stop() {
	if n.events != nil {
		close(n.quit)
		n.wg.Wait()
	}
}

// NotifyAll queues an Event for processing. If the queue is full, the event
// is dropped and logged to stderr.
func (n *Notifier) NotifyAll(e Event) {
	n.startOnce.Do(func() {
		n.events = make(chan Event, 100)
		n.wg.Add(1)
		go n.worker()
	})
	select {
	case n.events <- e:
	default:
		fmt.Fprintf(os.Stderr, "audit: event buffer full, dropping event\n")
	}
}

// worker reads events from the channel and dispatches them to all observers
// sequentially.
func (n *Notifier) worker() {
	defer n.wg.Done()
	for {
		select {
		case <-n.quit:
			return
		case event := <-n.events:
			n.mu.RLock()
			for _, o := range n.observers {
				o.Notify(event)
			}
			n.mu.RUnlock()
		}
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
	data, err := json.Marshal(e)
	if err != nil {
		fmt.Fprintf(os.Stderr, "audit: marshal error: %v\n", err)
		return
	}
	_, err = f.w.Write(append(data, '\n'))
	if err != nil {
		fmt.Fprintf(os.Stderr, "audit: write error: %v\n", err)
	}
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
	data, err := json.Marshal(e)
	if err != nil {
		fmt.Fprintf(os.Stderr, "audit: marshal error: %v\n", err)
		return
	}
	resp, err := h.client.Post(h.url, "application/json", bytes.NewReader(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "audit: http post error: %v\n", err)
		return
	}
	defer resp.Body.Close()
}
