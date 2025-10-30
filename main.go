package main

import (
	"context"
	"fmt"
	"tunnels/hub"
)

func main() {
	// defer atomic.AddInt32(&look, 1)
	// var lock server.SpinLock
	// var counter int

	// // 启动多个goroutine进行并发计数
	// for i := 0; i < 100; i++ {
	// 	go func() {
	// 		lock.Lock()
	// 		defer lock.Unlock()
	// 		counter++
	// 	}()
	// }

	// 等待所有goroutine完成（实际生产环境中应使用sync.WaitGroup）
	// time.Sleep(time.Second)
	// println(counter) // 应该输出100
	fmt.Println(1232132123)
	s := hub.NewHub(
		hub.WithAddress("0.0.0.0:9085"),
		hub.WithTimeout(1),
	)
	s.Start(context.Background())
}
