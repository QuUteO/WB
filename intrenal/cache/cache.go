package cache

import (
	model "WB_Service/intrenal/models"
	"sync"
	"time"
)

type Cache struct {
	mu      sync.RWMutex
	orders  map[string]*model.Order
	ttl     time.Duration
	expires map[string]time.Time
}

func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		orders:  make(map[string]*model.Order),
		ttl:     ttl,
		expires: make(map[string]time.Time),
	}
}

func (c *Cache) SetOrder(order *model.Order) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.orders[order.OrderUUID] = order
	c.expires[order.OrderUUID] = time.Now().Add(c.ttl)
}

func (c *Cache) GetOrder(orderUUID string) (*model.Order, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	order, exists := c.orders[orderUUID]
	if !exists {
		return nil, false
	}

	if time.Now().After(c.expires[order.OrderUUID]) {
		return nil, false
	}

	return order, true
}

func (c *Cache) GetAll() (map[string]*model.Order, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.orders) == 0 {
		return nil, false
	}

	result := make(map[string]*model.Order)
	now := time.Now()

	for uid, order := range c.orders {
		if now.Before(c.expires[uid]) {
			result[uid] = order
		}
	}

	if len(result) == 0 {
		return nil, false
	}

	return result, true
}

func (c *Cache) ReStoreCache(orders map[string]*model.Order) map[string]*model.Order {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.orders = orders
	now := time.Now()
	for uid := range c.orders {
		c.expires[uid] = now.Add(c.ttl)
	}
	return orders
}
