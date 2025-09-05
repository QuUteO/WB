package main

import (
	cache "WB_Service/intrenal/cache"
	"WB_Service/intrenal/config"
	"WB_Service/intrenal/db"
	"WB_Service/intrenal/http/handler"
	"WB_Service/intrenal/kafka/consumer"
	"WB_Service/intrenal/lib/sl"
	"WB_Service/intrenal/logger"
	model "WB_Service/intrenal/models"
	"WB_Service/intrenal/service"
	"errors"
	"fmt"
	"github.com/IBM/sarama"

	"context"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Контекст для управления жизненным циклом приложения
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// CONFIG
	cfg := config.MustLoad()

	// LOGGER
	log := logger.SetupLogger(cfg.Env)

	// DB + CACHE + SERVICE
	dbService, err := db.NewPostgres(ctx, cfg.Postgres, log)
	if err != nil {
		log.Error("failed to connect to database", sl.Err(err))
		os.Exit(1)
	}
	log.Info("connected to database")

	cacheService := cache.NewCache(cfg.TTl)
	// восстанавливаем кеш при перезапуске
	ordersss := cacheService.ReStoreCache(map[string]*model.Order{})
	fmt.Println(ordersss)

	orderService := service.NewService(dbService, cacheService, log)

	// инициализация producer
	producerCfg := sarama.NewConfig()
	producerCfg.Producer.RequiredAcks = sarama.WaitForAll // ждать подтверждения от всех реплик
	producerCfg.Producer.Retry.Max = 5                    // до 5 попыток при ошибке
	producerCfg.Producer.Return.Successes = true
	syncProducer, err := sarama.NewSyncProducer(cfg.Kafka.Brokers, producerCfg)

	// HTTP Router
	router := chi.NewRouter()
	handlers := serv.NewHandler(orderService, syncProducer)

	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	// API
	router.Get("/order/{order_uid}", handlers.GetOrder)
	router.Get("/orders", handlers.GetOrders)
	router.Post("/publish-order", handlers.PublishOrderHandler)

	// Статика (CSS, JS и т.п.)
	fs := http.FileServer(http.Dir("./static"))
	router.Handle("/static/*", http.StripPrefix("/static/", fs))

	// index.html на корне
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/index.html")
	})

	// HTTP server
	srv := &http.Server{
		Addr:         cfg.HTTPConfig.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPConfig.Timeout,
		WriteTimeout: cfg.HTTPConfig.Timeout,
		IdleTimeout:  cfg.HTTPConfig.IdleTimeout,
	}

	// Запуск Kafka consumer в отдельной горутине
	go func() {
		if err := consumer.StartConsumer(ctx, cfg, orderService); err != nil {
			log.Error("failed to start consumer", sl.Err(err))
			cancel()
		}
	}()

	// Запуск HTTP сервера в отдельной горутине
	go func() {
		log.Info("http server started", "address", cfg.HTTPConfig.Address)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("error starting HTTP server", sl.Err(err))
			cancel()
		}
	}()

	// Ожидание сигнала завершения
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-stop:
		log.Info("signal received, shutting down...")
	case <-ctx.Done():
		log.Info("context cancelled, shutting down...")
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("failed to gracefully shutdown server", sl.Err(err))
	} else {
		log.Info("http server stopped gracefully")
	}
}
