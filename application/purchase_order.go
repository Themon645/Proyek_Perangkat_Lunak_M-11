package application

import (
	"time"

	"m11/model"
	"m11/repository"
)

type PurchaseOrderService struct {
	repo *repository.PurchaseOrderRepository
}

func NewPurchaseOrderService(repo *repository.PurchaseOrderRepository) *PurchaseOrderService {
	return &PurchaseOrderService{repo: repo}
}

func (s *PurchaseOrderService) Create(po *model.PurchaseOrder) error {
	return s.repo.Create(po)
}

func (s *PurchaseOrderService) Receive(id string, received map[string]int) (*model.PurchaseOrder, error) {
	return s.repo.Receive(id, received)
}

func (s *PurchaseOrderService) Pay(id, proof string) (*model.PurchaseOrder, error) {
	return s.repo.Pay(id, proof)
}

func (s *PurchaseOrderService) Unpaid() ([]repository.UnpaidRow, int64, error) {
	return s.repo.Unpaid()
}

func (s *PurchaseOrderService) Report(from, to time.Time) (map[string]int64, error) {
	return s.repo.Report(from, to)
}
