package application

import (
	"time"

	"m11/dto"
	"m11/integration"
	"m11/model"
	"m11/repository"
)

type PurchaseOrderService struct {
	repo         *repository.PurchaseOrderRepository
	grnPublisher integration.GRNPublisher
}

func NewPurchaseOrderService(repo *repository.PurchaseOrderRepository, publisher ...integration.GRNPublisher) *PurchaseOrderService {
	service := &PurchaseOrderService{repo: repo}
	if len(publisher) > 0 {
		service.grnPublisher = publisher[0]
	}
	return service
}

func (s *PurchaseOrderService) Create(po *model.PurchaseOrder) error {
	return s.repo.Create(po)
}

func (s *PurchaseOrderService) Receive(id string, received map[string]int) (*model.PurchaseOrder, error) {
	return s.repo.Receive(id, received)
}

func (s *PurchaseOrderService) ReceiptHistory(id string) (*model.PurchaseOrder, []model.GoodsReceipt, error) {
	return s.repo.ReceiptHistory(id)
}

func (s *PurchaseOrderService) CreateGoodsReceipt(req *dto.CreateGoodsReceiptRequest, receivedDate time.Time) (*model.GoodsReceipt, error) {
	receipt, err := s.repo.CreateGoodsReceipt(req, receivedDate)
	if err == nil && s.grnPublisher != nil {
		s.grnPublisher.Publish(*receipt)
	}
	return receipt, err
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
