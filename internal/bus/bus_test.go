package bus

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPublishSubscribe(t *testing.T) {
	b := New(10)
	ch, cancel := b.Subscribe("inventory.changed", 4)
	defer cancel()

	b.Publish("inventory.changed", "pos", json.RawMessage(`{"sku":"x"}`))

	select {
	case m := <-ch:
		if m.Topic != "inventory.changed" || m.From != "pos" {
			t.Fatalf("unexpected message: %+v", m)
		}
	case <-time.After(time.Second):
		t.Fatal("no message received")
	}

	log := b.Log(10)
	if len(log) != 1 {
		t.Fatalf("expected 1 logged message, got %d", len(log))
	}
}

func TestWildcardSubscriber(t *testing.T) {
	b := New(10)
	ch, cancel := b.Subscribe("*", 4)
	defer cancel()
	b.Publish("a.b", "svc", nil)
	b.Publish("c.d", "svc", nil)
	count := 0
	timeout := time.After(time.Second)
	for count < 2 {
		select {
		case <-ch:
			count++
		case <-timeout:
			t.Fatalf("expected 2 wildcard messages, got %d", count)
		}
	}
}
