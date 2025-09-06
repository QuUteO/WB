package serv

import (
	model "WB_Service/intrenal/models"
	"context"
	"encoding/json"
	"github.com/IBM/sarama"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type OrderService interface {
	GetOrder(ctx context.Context, orderUID string) (*model.Order, error)
	GetOrders(ctx context.Context) (map[string]*model.Order, error)
	SaveOrder(ctx context.Context, order *model.Order) error
}

type Handler struct {
	service  OrderService
	producer sarama.SyncProducer
}

func NewHandler(service OrderService, producer sarama.SyncProducer) *Handler {
	return &Handler{
		service:  service,
		producer: producer,
	}
}

func (h *Handler) SaveOrder(ctx context.Context, order *model.Order) error {
	err := h.service.SaveOrder(ctx, order)
	if err != nil {
		return err
	}
	return nil
}

// GetOrderHandler GetOrder GET /orders/{order_uid}
func (h *Handler) GetOrderHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	orderUID := chi.URLParam(r, "order_uid")
	if orderUID == "" {
		http.Error(w, "order_uid is required", http.StatusBadRequest)
		return
	}

	order, err := h.service.GetOrder(ctx, orderUID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if order == nil {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(order)
}

// GetOrdersHandler GetOrders GET /orders
func (h *Handler) GetOrdersHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	orders, err := h.service.GetOrders(ctx)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// превращаем map[string]*Order в slice, чтобы красиво вернуть JSON
	resp := make([]*model.Order, 0, len(orders))
	for _, o := range orders {
		resp = append(resp, o)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// SaveOrderHandler PublishOrderHandler принимает JSON заказа и отправляет в Kafka
func (h *Handler) SaveOrderHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var order model.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// сериализуем обратно в []byte
	data, err := json.Marshal(order)
	if err != nil {
		http.Error(w, "failed to encode order: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// создаём Kafka сообщение
	msg := &sarama.ProducerMessage{
		Topic: "orders",
		Value: sarama.ByteEncoder(data),
		Key:   sarama.StringEncoder(order.OrderUUID),
	}

	partition, offset, err := h.producer.SendMessage(msg)
	if err != nil {
		http.Error(w, "failed to send to Kafka: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// отдаём результат
	resp := map[string]interface{}{
		"status":    "ok",
		"order_uid": order.OrderUUID,
		"partition": partition,
		"offset":    offset,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)

	_ = ctx // пока не используем, но можно, например, для логов
}
