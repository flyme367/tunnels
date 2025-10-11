package main

import (
	"context"
	"fmt"
	"tunnels/server"
)

var look int32 = 1

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
	s := server.NewServer(
		server.WithAddress("0.0.0.0:9085"),
		server.WithTimeout(1),
	)
	s.Start(context.Background())
}
