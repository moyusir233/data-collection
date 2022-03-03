package test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// 测试利用sync map能否保证仅进行一次注册
func TestSync_Map(t *testing.T) {
	count := 500
	var success int64 = 0
	m := new(sync.Map)
	ctx, cancel := context.WithCancel(context.Background())
	wg := new(sync.WaitGroup)
	wg.Add(count)
	register := func(key string) {
		defer wg.Done()
		<-ctx.Done()
		if _, ok := m.Load(key); !ok {
			m.Store(key, "")
			atomic.AddInt64(&success, 1)
		}
	}
	for i := 0; i < count; i++ {
		go register("test")
	}
	cancel()
	wg.Wait()
	if success != 1 {
		t.Error(success)
	}
}

// 测试reset ticker的开销
func BenchmarkResetTicker(b *testing.B) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for i := 0; i < b.N; i++ {
		t.Reset(time.Second)
	}
}
