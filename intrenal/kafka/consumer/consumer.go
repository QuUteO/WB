package consumer

import (
	"WB_Service/intrenal/config"
	serv "WB_Service/intrenal/http/handler"
	model "WB_Service/intrenal/models"
	"context"
	"encoding/json"
	"github.com/IBM/sarama"
	"github.com/go-playground/validator/v10"
	"log"
	"time"
)

var validate = validator.New()

type Consumer struct {
	OrderService serv.OrderService
}

func (c *Consumer) Setup(_ sarama.ConsumerGroupSession) error {
	log.Println("[Setup] Starting session KAFKA CONSUMER...")
	return nil
}

func (c *Consumer) Cleanup(_ sarama.ConsumerGroupSession) error {
	log.Println("[Cleanup] Cleaning up session...")
	return nil
}

func (c *Consumer) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var order model.Order
		if err := json.Unmarshal(msg.Value, &order); err != nil {
			log.Println("Kafka: bad message:", err)
			continue
		}

		// проверка на nil
		if order.Delivery == (model.Delivery{}) || order.Payment == (model.Payment{}) || len(order.Items) == 0 {
			log.Println("Kafka: order has empty required fields")
			continue
		}

		// валидируем
		if err := validate.Struct(&order); err != nil {
			log.Println("Kafka: struct validation failed:", err)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := c.OrderService.SaveOrder(ctx, &order); err != nil {
			log.Println("Kafka: save order failed:", err)
		}
		cancel()

		sess.MarkMessage(msg, "")
	}
	return nil
}

func subscribe(ctx context.Context, topic string, consumerGroup sarama.ConsumerGroup, service serv.OrderService) error {
	consumer := &Consumer{OrderService: service}
	go func() {
		for {
			if err := consumerGroup.Consume(ctx, []string{topic}, consumer); err != nil {
				log.Printf("Error from consumer: %v", err)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()
	return nil
}

func StartConsumer(ctx context.Context, cfg *config.Config, service serv.OrderService) error {

	saramaCfg := sarama.NewConfig()

	saramaCfg.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	saramaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest

	// создаем ConsumerGroup
	consumerGroup, err := sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.Kafka.GroupID, saramaCfg)
	if err != nil {
		log.Printf("Ошибка при создании consumer group: %s\n", err)
		return err
	}

	return subscribe(ctx, cfg.Kafka.Topic, consumerGroup, service)
}
