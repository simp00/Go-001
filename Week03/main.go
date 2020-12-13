package main

/*
#学号:G20200607011068
#班级:6
#作业链接:https://github.com/simp00/Go-001/tree/main/Week03
题目 基于 errgroup 实现一个 http server 的启动和关闭 ，以及 linux signal 信号的注册和处理，要保证能够 一个退出，全部注销退出。
*/
import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/sync/errgroup"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Server is a HTTP server.
type Server struct {
	srv *http.Server
}

// 创建http server，并模拟一个耗时接口
func NewServerRef() *Server {
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			//模拟耗时请求
			fmt.Println("receive request")
			time.Sleep(10 * time.Second)
		},
	))
	srv := &http.Server{
		Addr:    ":9801",
		Handler: mux,
	}
	return &Server{srv: srv}
}

func (s *Server) Start() error {
	fmt.Printf("[HTTP] Listening on: %s\n", s.srv.Addr)
	return s.srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func main() {
	stop := make(chan struct{})
	g, ctx := errgroup.WithContext(context.Background())
	svr := NewServerRef()
	//启动服务，当任何errorgroup中的goroutine产生error时，关闭httpServer
	g.Go(func() error {
		fmt.Println("start http")
		go func() {
			<-ctx.Done()
			fmt.Println("http ctx done")
			ctx2, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := svr.Shutdown(ctx2); err != nil {
				fmt.Println("Server forced to shutdown:", err)
			}
			stop <- struct{}{}
			fmt.Println("server exiting")
		}()
		return svr.Start()
	})

	//监听signal信号，当接收到退出相关信号退出
	g.Go(func() error {
		quit := make(chan os.Signal)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, os.Interrupt)
		for {
			fmt.Println("waiting for quit signal")
			select {
			case <-ctx.Done():
				fmt.Println("signal ctx done")
				return ctx.Err()
			case <-quit:
				return errors.New("receive quit signal")
			}
		}
	})

	//其他后台任务
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("background ctx done")
				return ctx.Err()
			default:
				fmt.Println("do something")
				time.Sleep(1 * time.Second)
			}
		}
	})
	err := g.Wait()
	fmt.Println(err)
	<-stop
	fmt.Println("server completely stopped!")
}
