package store

import (
	"context"
	"encoding/json"
	"net"
	"strconv"

	"github.com/segmentio/kafka-go"
)

// Events publishes domain events to Kafka.
type Events struct{ w *kafka.Writer }

// NewEvents makes a writer for the given brokers.
func NewEvents(brokers []string) *Events {
	return &Events{w: &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Balancer:               &kafka.Hash{},
		RequiredAcks:           kafka.RequireOne,
		AllowAutoTopicCreation: true,
		BatchSize:              1,
	}}
}

// Publish sends one event keyed by key.
func (e *Events) Publish(ctx context.Context, topic, key string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return e.w.WriteMessages(ctx, kafka.Message{Topic: topic, Key: []byte(key), Value: b})
}

// Close flushes and closes the writer.
func (e *Events) Close() { _ = e.w.Close() }

// EnsureTopics creates the topics orders-api publishes to, when missing: a
// first write to an auto-created topic fails while the broker creates it.
func EnsureTopics(ctx context.Context, brokers []string, topics ...string) error {
	conn, err := kafka.DialContext(ctx, "tcp", brokers[0])
	if err != nil {
		return err
	}
	defer conn.Close()
	ctrl, err := conn.Controller()
	if err != nil {
		return err
	}
	cc, err := kafka.DialContext(ctx, "tcp", net.JoinHostPort(ctrl.Host, strconv.Itoa(ctrl.Port)))
	if err != nil {
		return err
	}
	defer cc.Close()
	cfgs := make([]kafka.TopicConfig, 0, len(topics))
	for _, t := range topics {
		cfgs = append(cfgs, kafka.TopicConfig{Topic: t, NumPartitions: 1, ReplicationFactor: 1})
	}
	return cc.CreateTopics(cfgs...)
}
