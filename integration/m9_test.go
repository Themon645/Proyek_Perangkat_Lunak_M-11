package integration

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"m11/model"
)

func TestM9QueueRetriesUntilCreated(t *testing.T) {
	var attempts atomic.Int32
	done := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) < 3 {
			http.Error(w, "temporary failure", http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusCreated)
		close(done)
	}))
	defer server.Close()

	queue := NewM9Queue(server.URL, server.Client())
	queue.Start()
	queue.Publish(model.GoodsReceipt{ReceiptNo: "GR-1"})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("M9 queue did not complete retries")
	}
	if attempts.Load() != 3 {
		t.Fatalf("got %d attempts, want 3", attempts.Load())
	}
}
