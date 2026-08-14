package audit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"
)

type Event struct {
	Timestamp int64  `json:"ts"`
	Action    string `json:"action"`
	UserID    string `json:"user_id"`
	URL       string `json:"url"`
}

type Observer interface {
	Notify(Event)
}

type Notifier struct {
	observers []Observer
	mu        sync.Mutex
}

func NewNotifier() *Notifier { return &Notifier{} }

func (n *Notifier) Attach(o Observer) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.observers = append(n.observers, o)
}

func (n *Notifier) NotifyAll(e Event) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, o := range n.observers {
		go o.Notify(e)
	}
}

type FileReceiver struct {
	w *os.File
	m sync.Mutex
}

func NewFileReceiver(path string) (*FileReceiver, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileReceiver{w: f}, nil
}

func (f *FileReceiver) Notify(e Event) {
	f.m.Lock()
	defer f.m.Unlock()
	data, _ := json.Marshal(e)
	f.w.Write(append(data, '\n'))
}

type HTTPReceiver struct {
	url    string
	client *http.Client
}

func NewHTTPReceiver(url string) *HTTPReceiver {
	return &HTTPReceiver{url: url, client: &http.Client{Timeout: 10 * time.Second}}
}

func (h *HTTPReceiver) Notify(e Event) {
	data, _ := json.Marshal(e)
	resp, _ := h.client.Post(h.url, "application/json", bytes.NewReader(data))
	if resp != nil {
		resp.Body.Close()
	}
}
