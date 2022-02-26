package test

import (
	"testing"
	"time"
)

// 测试reset ticker的开销
func BenchmarkResetTicker(b *testing.B) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for i := 0; i < b.N; i++ {
		t.Reset(time.Second)
	}
}
