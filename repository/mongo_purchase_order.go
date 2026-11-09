package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"m11/common"
	"m11/common/validation"
	"m11/dto"
	"m11/model"
)

// NewMongoPurchaseOrderRepository membuat repository PO dengan backend MongoDB.
func NewMongoPurchaseOrderRepository(mongoDB *mongo.Database) *PurchaseOrderRepository {
	return &PurchaseOrderRepository{mongoDB: mongoDB}
}

func (r *PurchaseOrderRepository) createMongo(po *model.PurchaseOrder) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	col := r.mongoDB.Collection("purchase_orders")
	count, err := col.CountDocuments(ctx, bson.M{"_id": po.ID})
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("purchase order already exists")
	}

	now := time.Now().UTC()
	if po.UUID == uuid.Nil {
		po.UUID = uuid.New()
	}
	po.CreatedAt = now
	po.UpdatedAt = now
	if po.Payment == "" {
		po.Payment = common.PaymentUnpaid
	}

	for i := range po.Items {
		if po.Items[i].UUID == uuid.Nil {
			po.Items[i].UUID = uuid.New()
		}
		po.Items[i].PurchaseOrderUUID = po.UUID
		po.Items[i].CreatedAt = now
		po.Items[i].UpdatedAt = now
	}

	_, err = col.InsertOne(ctx, po)
	return err
}

func (r *PurchaseOrderRepository) findMongo(id string) (*model.PurchaseOrder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var po model.PurchaseOrder
	err := r.mongoDB.Collection("purchase_orders").FindOne(ctx, bson.M{"_id": id}).Decode(&po)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &po, nil
}

func (r *PurchaseOrderRepository) listMongo() ([]model.PurchaseOrder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	findOpts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})
	cursor, err := r.mongoDB.Collection("purchase_orders").Find(ctx, bson.M{}, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var orders []model.PurchaseOrder
	if err := cursor.All(ctx, &orders); err != nil {
		return nil, err
	}
	if orders == nil {
		orders = []model.PurchaseOrder{}
	}
	return orders, nil
}

func (r *PurchaseOrderRepository) receiptHistoryMongo(id string) (*model.PurchaseOrder, []model.GoodsReceipt, error) {
	po, err := r.findMongo(id)
	if err != nil {
		return nil, nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	findOpts := options.Find().SetSort(bson.D{{Key: "received_date", Value: 1}, {Key: "created_at", Value: 1}})
	cursor, err := r.mongoDB.Collection("goods_receipts").Find(ctx, bson.M{"purchase_order_id": po.ID}, findOpts)
	if err != nil {
		return nil, nil, err
	}
	defer cursor.Close(ctx)

	var receipts []model.GoodsReceipt
	if err := cursor.All(ctx, &receipts); err != nil {
		return nil, nil, err
	}

	// Buat map items dari PO untuk melengkapi data item name
	itemMap := make(map[string]model.PurchaseOrderItem, len(po.Items))
	for _, it := range po.Items {
		itemMap[it.ID] = it
	}

	for i := range receipts {
		for j := range receipts[i].Items {
			if it, ok := itemMap[receipts[i].Items[j].Item.ID]; ok {
				receipts[i].Items[j].Item.Name = it.Name
			}
		}
	}

	if receipts == nil {
		receipts = []model.GoodsReceipt{}
	}
	return po, receipts, nil
}

func (r *PurchaseOrderRepository) receiveMongo(id string, received map[string]int) (*model.PurchaseOrder, error) {
	po, err := r.findMongo(id)
	if err != nil {
		return nil, err
	}
	if po.Status != common.StatusApproved && po.Status != common.StatusReceivedPartial {
		return nil, errors.New("purchase order must be approved")
	}

	items := make(map[string]*model.PurchaseOrderItem, len(po.Items))
	for i := range po.Items {
		items[po.Items[i].ID] = &po.Items[i]
	}

	for itemID, qty := range received {
		item, ok := items[itemID]
		if !ok || qty <= 0 {
			return nil, errors.New("received quantity exceeds ordered quantity")
		}
		if err := validation.ValidatePartialQuantity(item.Quantity, item.Received, qty); err != nil {
			return nil, err
		}
	}

	for itemID, qty := range received {
		items[itemID].Received += qty
	}

	complete := true
	for _, item := range po.Items {
		if item.Received < item.Quantity {
			complete = false
			break
		}
	}

	po.Status = common.StatusReceivedPartial
	if complete {
		po.Status = common.StatusReceived
	}
	now := time.Now().UTC()
	po.ReceivedAt = &now
	po.UpdatedAt = now

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = r.mongoDB.Collection("purchase_orders").UpdateOne(
		ctx,
		bson.M{"_id": po.ID},
		bson.M{"$set": bson.M{
			"items":       po.Items,
			"status":      po.Status,
			"received_at": po.ReceivedAt,
			"updated_at":  po.UpdatedAt,
		}},
	)
	if err != nil {
		return nil, err
	}

	return po, nil
}

func (r *PurchaseOrderRepository) createGoodsReceiptMongo(req *dto.CreateGoodsReceiptRequest, receivedDate time.Time) (*model.GoodsReceipt, error) {
	po, err := r.findMongo(req.PurchaseOrderID)
	if err != nil {
		return nil, err
	}
	if po.Status != common.StatusApproved && po.Status != common.StatusReceivedPartial {
		return nil, errors.New("purchase order must be approved")
	}

	items := make(map[string]*model.PurchaseOrderItem, len(po.Items))
	for i := range po.Items {
		items[po.Items[i].ID] = &po.Items[i]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Hitung total penerimaan per item dari goods_receipts sebelumnya
	cursor, err := r.mongoDB.Collection("goods_receipts").Find(ctx, bson.M{"purchase_order_id": po.ID})
	if err != nil {
		return nil, err
	}
	var existingReceipts []model.GoodsReceipt
	_ = cursor.All(ctx, &existingReceipts)
	cursor.Close(ctx)

	totals := make(map[string]int)
	for _, rc := range existingReceipts {
		for _, it := range rc.Items {
			totals[it.Item.ID] += it.QtyReceived
		}
	}

	now := time.Now().UTC()
	receiptUUID := uuid.New()
	receipt := model.GoodsReceipt{
		ReceiptNo:         req.ReceiptNo,
		PurchaseOrderID:   po.ID,
		PurchaseOrderUUID: po.UUID,
		ReceivedDate:      receivedDate,
		Receiver:          req.Receiver,
		Notes:             req.Notes,
		Items:             make([]model.GoodsReceiptItem, 0, len(req.Items)),
	}
	receipt.AuditFields.UUID = receiptUUID
	receipt.AuditFields.CreatedAt = now
	receipt.AuditFields.UpdatedAt = now

	for _, item := range req.Items {
		orderItem, ok := items[item.ItemID]
		if !ok {
			return nil, errors.New("received quantity exceeds ordered quantity")
		}
		if err := validation.ValidatePartialQuantity(orderItem.Quantity, totals[orderItem.ID], item.QtyReceived); err != nil {
			return nil, err
		}
		receipt.Items = append(receipt.Items, model.GoodsReceiptItem{
			ID:          uuid.NewString(),
			ReceiptUUID: receiptUUID,
			QtyReceived: item.QtyReceived,
			Item: model.PurchaseOrderItem{
				ID:   orderItem.ID,
				Name: orderItem.Name,
			},
		})
		totals[orderItem.ID] += item.QtyReceived
	}

	// Update items di PO
	complete := true
	for i := range po.Items {
		po.Items[i].Received = totals[po.Items[i].ID]
		if po.Items[i].Received < po.Items[i].Quantity {
			complete = false
		}
	}

	po.Status = common.StatusReceivedPartial
	if complete {
		po.Status = common.StatusReceived
	}
	po.UpdatedAt = now

	// Simpan receipt baru
	if _, err := r.mongoDB.Collection("goods_receipts").InsertOne(ctx, receipt); err != nil {
		return nil, err
	}

	// Update PO di MongoDB
	_, err = r.mongoDB.Collection("purchase_orders").UpdateOne(
		ctx,
		bson.M{"_id": po.ID},
		bson.M{"$set": bson.M{
			"items":      po.Items,
			"status":     po.Status,
			"updated_at": po.UpdatedAt,
		}},
	)
	if err != nil {
		return nil, err
	}

	return &receipt, nil
}

func (r *PurchaseOrderRepository) payMongo(id, proof string) (*model.PurchaseOrder, error) {
	po, err := r.findMongo(id)
	if err != nil {
		return nil, err
	}
	if po.Status != common.StatusReceived && po.Status != common.StatusReceivedPartial {
		return nil, errors.New("purchase order must be received before payment")
	}
	if po.Payment == common.PaymentPaid {
		return nil, errors.New("purchase order sudah dibayar")
	}

	var amount int64
	for _, item := range po.Items {
		amount += int64(item.Received) * item.UnitCost
	}

	now := time.Now().UTC()
	po.Payment = common.PaymentPaid
	po.Proof = proof
	po.UpdatedAt = now

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = r.mongoDB.Collection("purchase_orders").UpdateOne(
		ctx,
		bson.M{"_id": po.ID},
		bson.M{"$set": bson.M{
			"payment":    po.Payment,
			"proof":      po.Proof,
			"updated_at": po.UpdatedAt,
		}},
	)
	if err != nil {
		return nil, err
	}

	// Simpan riwayat payment record
	paymentRec := model.PaymentRecord{
		PurchaseOrderID:   po.ID,
		PurchaseOrderUUID: po.UUID,
		Amount:            amount,
		PaidAt:            now,
		ProofPath:         proof,
	}
	paymentRec.AuditFields.UUID = uuid.New()
	paymentRec.AuditFields.CreatedAt = now
	paymentRec.AuditFields.UpdatedAt = now

	_, _ = r.mongoDB.Collection("payment_records").InsertOne(ctx, paymentRec)

	return po, nil
}

func (r *PurchaseOrderRepository) unpaidMongo() ([]UnpaidRow, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{
		"payment": common.PaymentUnpaid,
		"status":  bson.M{"$in": []string{common.StatusReceived, common.StatusReceivedPartial}},
	}
	cursor, err := r.mongoDB.Collection("purchase_orders").Find(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var orders []model.PurchaseOrder
	if err := cursor.All(ctx, &orders); err != nil {
		return nil, 0, err
	}

	rows := make([]UnpaidRow, 0, len(orders))
	var total int64
	for _, po := range orders {
		var amount int64
		for _, item := range po.Items {
			amount += int64(item.Received) * item.UnitCost
		}
		rows = append(rows, UnpaidRow{ID: po.ID, Supplier: po.Supplier, Total: amount})
		total += amount
	}

	return rows, total, nil
}

func (r *PurchaseOrderRepository) reportMongo(from, to time.Time) (map[string]int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{
		"received_at": bson.M{
			"$gte": from,
			"$lt":  to.Add(24 * time.Hour),
		},
	}
	cursor, err := r.mongoDB.Collection("purchase_orders").Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var orders []model.PurchaseOrder
	if err := cursor.All(ctx, &orders); err != nil {
		return nil, err
	}

	totals := map[string]int64{}
	for _, po := range orders {
		for _, item := range po.Items {
			totals[po.Supplier] += int64(item.Received) * item.UnitCost
		}
	}

	return totals, nil
}

