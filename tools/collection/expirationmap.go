package collection

import (
	"time"
)

const (
	defaultCleanDuration = time.Minute * 5
)

type expirationMap[K comparable, V any] struct {
	ConcurrentMap[K, V]
	expired       ConcurrentMap[K, time.Time]
	cleanDuration time.Duration
	stop          chan struct{}
}

func NewExpirationMap[K comparable, V any](opts ...ExpirationMapOption[K, V]) ExpirationMap[K, V] {
	m := &expirationMap[K, V]{
		expired: NewConcurrentMap[K, time.Time](),
		stop:    make(chan struct{}),
	}
	for _, opt := range opts {
		opt(m)
	}

	if m.ConcurrentMap == nil {
		m.ConcurrentMap = NewConcurrentMap[K, V]()
	}

	if m.cleanDuration == 0 {
		m.cleanDuration = defaultCleanDuration
	}

	go m.cleaner()

	return m
}

type ExpirationMapOption[K comparable, V any] func(c *expirationMap[K, V])

func WithConcurrentMap[K comparable, V any](cm ConcurrentMap[K, V]) ExpirationMapOption[K, V] {
	return func(c *expirationMap[K, V]) {
		c.ConcurrentMap = cm
	}
}

func WithCleanDuration[K comparable, V any](duration time.Duration) ExpirationMapOption[K, V] {
	if duration <= 0 {
		duration = defaultCleanDuration
	}
	return func(c *expirationMap[K, V]) {
		c.cleanDuration = duration
	}
}

func (e *expirationMap[K, V]) SetExpired(key K, val V, duration time.Duration) {
	if duration < 0 {
		panic("negative duration")
	}
	if duration == 0 {
		return
	}

	e.expired.Set(key, time.Now().Add(duration))
	e.Set(key, val)
}

func (e *expirationMap[K, V]) Get(key K) (V, bool) {
	val, ok := e.Get(key)
	if !ok {
		return *new(V), false
	}

	exp, ok1 := e.expired.Get(key)
	if !ok1 || exp.Before(time.Now()) {
		e.Delete(key)
		e.expired.Delete(key)
		return *new(V), false
	}

	return val, true
}

func (e *expirationMap[K, V]) Destroy() {
	close(e.stop)
}

func (e *expirationMap[K, V]) cleaner() {
	ticker := time.NewTicker(e.cleanDuration)
	defer ticker.Stop()
	for {
		select {
		case <-e.stop:
			return
		case <-ticker.C:
			keys := e.expired.Keys()
			for _, key := range keys {
				v, ok := e.expired.Get(key)
				if !ok || v.After(time.Now()) {
					continue
				}

				e.Delete(key)
				e.expired.Delete(key)
			}
		}
	}
}
