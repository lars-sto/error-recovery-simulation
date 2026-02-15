package adapter

import (
	"sync"

	"github.com/pion/interceptor/pkg/flexfec"
)

// RuntimeBus is an in-process ConfigSource that allows pushing runtime configs
// from an external policy engine into the interceptor
type RuntimeBus struct {
	mu sync.Mutex
	cb map[uint32]func(flexfec.RuntimeConfig)
}

func NewRuntimeBus() *RuntimeBus {
	return &RuntimeBus{cb: make(map[uint32]func(flexfec.RuntimeConfig))}
}

func (b *RuntimeBus) Subscribe(key flexfec.StreamKey, fn func(cfg flexfec.RuntimeConfig)) (unsubscribe func()) {
	b.mu.Lock()
	b.cb[key.MediaSSRC] = fn
	b.mu.Unlock()

	return func() {
		b.mu.Lock()
		delete(b.cb, key.MediaSSRC)
		b.mu.Unlock()
	}
}

func (b *RuntimeBus) Publish(mediaSSRC uint32, cfg flexfec.RuntimeConfig) {
	b.mu.Lock()
	fn := b.cb[mediaSSRC]
	b.mu.Unlock()
	if fn != nil {
		fn(cfg)
	}
}
