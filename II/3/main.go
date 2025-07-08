package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type OrderRequest struct {
	ClientRequestID string `json:"clientRequestId"`
}

type OrderResponse struct {
	OrderID string `json:"orderId"`
	Status  string `json:"status"` // "new" or "duplicate"
}

type OrderService struct {
	mu    sync.Mutex
	cache map[string]string // clientRequestId -> orderId
}

// 模拟创建订单（生成随机 ID）
func generateOrderID() string {
	return fmt.Sprintf("ORD-%d", rand.Intn(1000000))
}

func (s *OrderService) HandleOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ClientRequestID == "" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否已处理该 clientRequestId
	if existingOrderID, ok := s.cache[req.ClientRequestID]; ok {
		resp := OrderResponse{OrderID: existingOrderID, Status: "duplicate"}
		json.NewEncoder(w).Encode(resp)
		return
	}

	// 创建新订单
	orderID := generateOrderID()
	s.cache[req.ClientRequestID] = orderID

	resp := OrderResponse{OrderID: orderID, Status: "new"}
	json.NewEncoder(w).Encode(resp)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	service := &OrderService{
		cache: make(map[string]string),
	}

	http.HandleFunc("/order", service.HandleOrder)
	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
