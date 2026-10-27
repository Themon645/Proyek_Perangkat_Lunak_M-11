package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"m11/model"
)

type GRNPublisher interface {
	Publish(model.GoodsReceipt)
}

type M9Queue struct {
	endpoint string
	client   *http.Client
	jobs     chan model.GoodsReceipt
}

func NewM9Queue(endpoint string, client *http.Client) *M9Queue {
	if client == nil {
		client = http.DefaultClient
	}
	return &M9Queue{
		endpoint: endpoint,
		client:   client,
		jobs:     make(chan model.GoodsReceipt, 32),
	}
}

func (q *M9Queue) Start() {
	go func() {
		for receipt := range q.jobs {
			q.send(receipt)
		}
	}()
}

func (q *M9Queue) Publish(receipt model.GoodsReceipt) {
	if q.endpoint == "" {
		log.Printf("M9 GRN sync skipped: M9_GRN_ENDPOINT is empty")
		return
	}
	q.jobs <- receipt
}

func (q *M9Queue) send(receipt model.GoodsReceipt) {
	body, err := json.Marshal(receipt)
	if err != nil {
		log.Printf("M9 GRN sync marshal failed for %s: %v", receipt.ReceiptNo, err)
		return
	}
	for attempt := 1; attempt <= 3; attempt++ {
		req, err := http.NewRequest(http.MethodPost, q.endpoint, bytes.NewReader(body))
		if err != nil {
			log.Printf("M9 GRN sync request failed for %s: %v", receipt.ReceiptNo, err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		res, err := q.client.Do(req)
		if err == nil {
			res.Body.Close()
			if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusCreated {
				log.Printf("M9 GRN sync succeeded for %s: HTTP %d", receipt.ReceiptNo, res.StatusCode)
				return
			}
			err = fmt.Errorf("HTTP %d", res.StatusCode)
		}
		if attempt == 3 {
			log.Printf("M9 GRN sync failed for %s after 3 attempts: %v", receipt.ReceiptNo, err)
			return
		}
		time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
	}
}
