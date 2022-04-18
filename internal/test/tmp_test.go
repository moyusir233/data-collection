package test

import (
	"context"
	v1 "gitee.com/moyusir/data-collection/api/dataCollection/v1"
	util "gitee.com/moyusir/util/api/util/v1"
	"reflect"
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

// 测试利用反射提取结构体值的开销
func BenchmarkReflect(b *testing.B) {
	state := &util.TestedDeviceState{
		Id:          "test",
		Voltage:     123,
		Current:     123,
		Temperature: 123,
	}
	fields := []string{"Voltage", "Current", "Temperature"}
	for i := 0; i < b.N; i++ {
		value := reflect.ValueOf(*state)
		for _, f := range fields {
			value.FieldByName(f).Float()
		}
	}
}

func TestMaxConnectionTime(t *testing.T) {
	// 初始化测试所需环境变量
	envs := map[string]string{
		"USERNAME": "init_bucket",
	}
	bootstrap, err := generalInit("", envs)
	if err != nil {
		t.Fatal(err)
	}

	configClient, _, warningDetectClient := StartDataCollectionTestServer(t, bootstrap)

	wg := new(sync.WaitGroup)
	ctx, cancelFunc := context.WithCancel(context.Background())
	once := new(sync.Once)
	cancel := func() {
		once.Do(func() {
			cancelFunc()
		})
	}

	timeout := time.After(3 * time.Minute)
	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})

	configUpdateStream, err := configClient.CreateConfigUpdateStream0(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, err = configUpdateStream.Header()
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		wg.Add(1)
		defer wg.Done()

		defer configUpdateStream.Send(&v1.ConfigUpdateReply{
			Success: true,
			End:     true,
		})
		go func() {
			for {
				_, err := configUpdateStream.Recv()
				if err != nil {
					select {
					case <-ctx.Done():
					default:
						t.Error(err)
						cancel()
					}
					return
				}
			}
		}()
		<-ctx.Done()
	}()

	configSaveStream, err := configClient.CreateInitialConfigSaveStream0(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer configSaveStream.CloseSend()

	stateInfoSaveStream, err := warningDetectClient.CreateStateInfoSaveStream0(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer stateInfoSaveStream.CloseSend()

	select {
	case <-timeout:
		return
	case <-ctx.Done():
		t.Fatal("连接被提前断开了")
	}
}
