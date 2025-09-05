package service

import (
	"WB_Service/intrenal/cache"
	"WB_Service/intrenal/db"
	"WB_Service/intrenal/lib/sl"
	model "WB_Service/intrenal/models"
	"context"
	"log/slog"
)

type Service struct {
	db    *db.Postgres
	cache *cache.Cache
	log   *slog.Logger
}

func NewService(db *db.Postgres, cache *cache.Cache, log *slog.Logger) *Service {
	return &Service{
		db:    db,
		cache: cache,
		log:   log,
	}
}

func (s *Service) SaveOrder(ctx context.Context, order *model.Order) error {
	// сохраняем в БД
	if err := s.db.SaveUserData(ctx, order); err != nil {
		s.log.Error("Error saving order", sl.Err(err))
		return err
	}

	// сохраняеем в кеш
	s.cache.SetOrder(order)

	s.log.Info("Save order successfully")
	return nil
}

func (s *Service) GetOrder(ctx context.Context, orderUID string) (*model.Order, error) {
	// сначала берем из кеша
	if order, found := s.cache.GetOrder(orderUID); found {
		s.log.Info("Get order successfully")
		return order, nil
	}

	// если не нашли в кеше ищем в базе данных
	order, err := s.db.GetOrder(ctx, orderUID)
	if err != nil {
		s.log.Error("Error saving order", sl.Err(err))
		return nil, err
	}
	if order != nil {
		// сохраняем в кеш для будущим запросов
		s.cache.SetOrder(order)
		s.log.Info("Order found in DB and cache successfully")
	}

	return order, nil
}

func (s *Service) GetOrders(ctx context.Context) (map[string]*model.Order, error) {
	// сначала пробуем из кеша
	if orders, found := s.cache.GetAll(); found {
		s.log.Info("Get all orders from cache")
		return orders, nil
	}

	// если в кеше нет — идём в БД
	orders, err := s.db.GetOrders(ctx)
	if err != nil {
		s.log.Error("Error getting orders from DB", sl.Err(err))
		return nil, err
	}

	// восстанавливаем кеш из БД
	s.cache.ReStoreCache(orders)

	s.log.Info("Get all orders from DB and restored cache")
	return orders, nil
}

func (s *Service) RestoreCache(ctx context.Context) error {
	// получаем все значения из базы данных
	orders, err := s.db.GetOrders(ctx)
	if err != nil {
		s.log.Error("Error get orders", sl.Err(err))
		return err
	}

	s.cache.ReStoreCache(orders)
	s.log.Info("Restore cache successfully")

	return nil
}
