package ws

import (
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

const testDelay = 50 * time.Millisecond

type saveCall struct{ docID, content string }

// recorder faz o papel do banco: guarda cada gravação feita.
type recorder struct {
	mu    sync.Mutex
	calls []saveCall
}

func (r *recorder) save(docID, content string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, saveCall{docID, content})
	return nil
}

func (r *recorder) get() []saveCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]saveCall(nil), r.calls...)
}

// newSavers cria n autosavers (um por réplica) no mesmo Redis, gravando
// todos no mesmo recorder.
func newSavers(t *testing.T, n int) ([]*Autosaver, *recorder, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rec := &recorder{}
	savers := make([]*Autosaver, n)
	for i := range savers {
		// Sem retries: com o Redis fora, o erro vem na hora (e não depois do fim do teste).
		bus := NewRedisBus(redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1}))
		savers[i] = NewAutosaver(bus, rec.save, testDelay)
	}
	return savers, rec, mr
}

func edit(id, docID, content string) Event {
	return Event{ID: id, DocumentID: docID, Type: "edit", Payload: content}
}

func TestAutosaverDebouncesToLastContent(t *testing.T) {
	savers, rec, _ := newSavers(t, 1)
	for i, content := range []string{"o", "ol", "olá"} {
		savers[0].Touch(edit(string(rune('a'+i)), "doc-1", content))
		time.Sleep(testDelay / 5) // digitação: cada evento antes do timer vencer
	}

	if got := rec.get(); len(got) != 0 {
		t.Fatalf("gravou antes do debounce: %v", got)
	}
	time.Sleep(3 * testDelay)

	got := rec.get()
	if len(got) != 1 || got[0] != (saveCall{"doc-1", "olá"}) {
		t.Fatalf("gravações %v, esperava uma com o último conteúdo", got)
	}
}

func TestAutosaverSingleWriteAcrossReplicas(t *testing.T) {
	// As duas réplicas recebem os mesmos eventos pelo Pub/Sub.
	savers, rec, _ := newSavers(t, 2)
	for _, e := range []Event{edit("e1", "doc-1", "a"), edit("e2", "doc-1", "ab")} {
		for _, s := range savers {
			s.Touch(e)
		}
	}
	time.Sleep(3 * testDelay)

	got := rec.get()
	if len(got) != 1 || got[0] != (saveCall{"doc-1", "ab"}) {
		t.Fatalf("gravações %v, esperava exatamente uma com \"ab\"", got)
	}
}

func TestAutosaverSavesEachPause(t *testing.T) {
	// Duas rajadas de edição no mesmo documento: cada pausa gera uma gravação.
	savers, rec, _ := newSavers(t, 1)
	savers[0].Touch(edit("e1", "doc-1", "primeira"))
	time.Sleep(3 * testDelay)
	savers[0].Touch(edit("e2", "doc-1", "segunda"))
	time.Sleep(3 * testDelay)

	got := rec.get()
	if len(got) != 2 || got[1].content != "segunda" {
		t.Fatalf("gravações %v, esperava duas (uma por pausa)", got)
	}
}

func TestAutosaverDocumentsAreIndependent(t *testing.T) {
	savers, rec, _ := newSavers(t, 1)
	savers[0].Touch(edit("e1", "doc-1", "um"))
	savers[0].Touch(edit("e2", "doc-2", "dois"))
	time.Sleep(3 * testDelay)

	got := rec.get()
	if len(got) != 2 {
		t.Fatalf("gravações %v, esperava uma por documento", got)
	}
}

func TestAutosaverSkipsWhenRedisIsDown(t *testing.T) {
	mr := miniredis.RunT(t)
	rec := &recorder{}
	// DialTimeout curto: no Windows, conectar numa porta fechada leva ~2s.
	client := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1, DialTimeout: 100 * time.Millisecond})
	saver := NewAutosaver(NewRedisBus(client), rec.save, testDelay)
	mr.Close()
	saver.Touch(edit("e1", "doc-1", "x"))
	time.Sleep(testDelay + 500*time.Millisecond) // o lock já falhou (e a gravação teria acontecido)

	if got := rec.get(); len(got) != 0 {
		t.Fatalf("gravou sem conseguir o lock: %v", got)
	}
}
