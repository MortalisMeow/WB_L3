package repo

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"delayed-notifier/internal/domain"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	delayedQueue       = "notifications.delayed"
	mainQueue          = "notifications.main"
	deadLetterExchange = "notifications.dlx"
)

type RabbitMQRepository struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitMQRepository(url string) (*RabbitMQRepository, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("channel: %w", err)
	}

	repo := &RabbitMQRepository{conn: conn, channel: channel}

	if err := repo.setup(); err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("setup: %w", err)
	}

	log.Println("[RABBITMQ] Connected")
	return repo, nil
}

func (r *RabbitMQRepository) setup() error {
	if err := r.channel.ExchangeDeclare(deadLetterExchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("DLX: %w", err)
	}

	if _, err := r.channel.QueueDeclare(mainQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("main queue: %w", err)
	}

	args := amqp.Table{
		"x-dead-letter-exchange":    deadLetterExchange,
		"x-dead-letter-routing-key": mainQueue,
	}
	if _, err := r.channel.QueueDeclare(delayedQueue, true, false, false, false, args); err != nil {
		return fmt.Errorf("delayed queue: %w", err)
	}

	if err := r.channel.QueueBind(mainQueue, mainQueue, deadLetterExchange, false, nil); err != nil {
		return fmt.Errorf("bind: %w", err)
	}

	return nil
}

func (r *RabbitMQRepository) Publish(n domain.Notification, delay time.Duration) error {
	body, err := json.Marshal(n)
	if err != nil {
		return err
	}

	return r.channel.Publish(
		"", delayedQueue, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Expiration:   fmt.Sprintf("%d", delay.Milliseconds()),
		},
	)
}

func (r *RabbitMQRepository) Consume(handler func(domain.Notification) error) error {
	msgs, err := r.channel.Consume(mainQueue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	log.Println("[RABBITMQ] Consumer started")

	go func() {
		for msg := range msgs {
			var n domain.Notification
			if err := json.Unmarshal(msg.Body, &n); err != nil {
				log.Printf("[RABBITMQ] Unmarshal error: %v", err)
				msg.Nack(false, false)
				continue
			}

			log.Printf("[RABBITMQ] Processing %s (retry: %d)", n.ID, n.RetryCount)

			if err := handler(n); err != nil {
				log.Printf("[RABBITMQ] Error, will retry: %v", err)
				msg.Nack(false, true) // requeue
			} else {
				msg.Ack(false)
			}
		}
	}()

	return nil
}

func (r *RabbitMQRepository) Close() {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
	log.Println("[RABBITMQ] Closed")
}
