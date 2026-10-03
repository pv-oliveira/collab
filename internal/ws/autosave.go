package ws

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Autosaver persiste o documento depois de delay sem edições. Roda em todas
// as réplicas, com inscrição própria no canal: como todas veem a mesma
// sequência de eventos, todas disputam o lock do mesmo último evento e só
// uma grava — sempre o conteúdo mais novo, e mesmo que a de origem caia.
type Autosaver struct {
	bus   *RedisBus
	save  func(docID, content string) error
	delay time.Duration

	mu     sync.Mutex
	timers map[string]*time.Timer // por documento
}

func NewAutosaver(bus *RedisBus, save func(docID, content string) error, delay time.Duration) *Autosaver {
	return &Autosaver{bus: bus, save: save, delay: delay, timers: make(map[string]*time.Timer)}
}

// Run se inscreve no canal e retorna; o debounce segue nos timers.
func (a *Autosaver) Run(ctx context.Context) error {
	return a.bus.Subscribe(ctx, a.Touch)
}

// Touch reinicia o debounce do documento com o evento mais recente.
func (a *Autosaver) Touch(event Event) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if t := a.timers[event.DocumentID]; t != nil {
		t.Stop()
	}
	var timer *time.Timer
	// O callback pega o mu antes de ler timer: só roda depois da atribuição.
	timer = time.AfterFunc(a.delay, func() {
		a.mu.Lock()
		if a.timers[event.DocumentID] == timer {
			delete(a.timers, event.DocumentID)
		}
		a.mu.Unlock()
		a.flush(event)
	})
	a.timers[event.DocumentID] = timer
}

func (a *Autosaver) flush(event Event) {
	ctx := context.Background()
	err := a.bus.client.SetArgs(ctx, "autosave:"+event.ID, 1, redis.SetArgs{Mode: "NX", TTL: time.Minute}).Err()
	if errors.Is(err, redis.Nil) {
		return // NX não gravou: outra réplica já pegou este evento
	}
	if err != nil {
		log.Println("autosave: lock:", err)
		return
	}
	if err := a.save(event.DocumentID, event.Payload); err != nil {
		log.Println("autosave:", err)
	}
}
