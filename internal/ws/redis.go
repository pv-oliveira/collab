package ws

import (
	"context"
	"encoding/json"
	"log"

	"github.com/redis/go-redis/v9"
)

// Canal único: todas as instâncias recebem todos os eventos e filtram por documento.
const eventsChannel = "collab:events"

type RedisBus struct {
	client *redis.Client
}

func NewRedisBus(ctx context.Context, url string) (*RedisBus, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisBus{client: client}, nil
}

func (r *RedisBus) Publish(ctx context.Context, event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return r.client.Publish(ctx, eventsChannel, data).Err()
}

// Subscribe só retorna depois que a inscrição foi confirmada pelo Redis,
// para nenhum evento publicado logo em seguida ser perdido.
func (r *RedisBus) Subscribe(ctx context.Context, handler func(Event)) error {
	sub := r.client.Subscribe(ctx, eventsChannel)
	if _, err := sub.Receive(ctx); err != nil {
		return err
	}

	go func() {
		for msg := range sub.Channel() {
			var event Event
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				log.Println("ws: evento inválido no redis:", err)
				continue
			}
			handler(event)
		}
	}()

	return nil
}
