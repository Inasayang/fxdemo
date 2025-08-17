package main

import (
	"context"
	"io"
	"net"
	"net/http"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// 定义一个 Route 类型。这是 http.Handler 的扩展，处理程序知道自己的注册路径。
type Route interface {
	http.Handler

	// Pattern reports the path at which this is registered.
	Pattern() string
}

type EchoHandler struct {
	log *zap.Logger
}

func NewEchoHandler(log *zap.Logger) *EchoHandler {
	return &EchoHandler{log: log}
}

// 修改 EchoHandler 以实现该接口。
func (*EchoHandler) Pattern() string {
	return "/echo"
}

func (h *EchoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, err := io.Copy(w, r.Body); err != nil {
		h.log.Warn("Failed to handle request", zap.Error(err))
	} else {
		h.log.Info("Request handled successfully", zap.String("method", r.Method), zap.String("url", r.URL.String()))
	}
}

// 修改 NewServeMux 以接受一个 Route 并使用其提供的模式。
func NewServeMux(route Route) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle(route.Pattern(), route)
	return mux
}

func NewHTTPServer(lc fx.Lifecycle, mux *http.ServeMux, log *zap.Logger) *http.Server {
	srv := &http.Server{Addr: ":8080", Handler: mux}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			log.Info("Starting HTTP server", zap.String("addr", srv.Addr))
			go srv.Serve(ln)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}

func main() {

	fx.New(
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
		fx.Provide(
			NewHTTPServer,
			NewServeMux,
			// 注释 NewEchoHandler 条目，说明处理程序应作为 Route 提供。
			fx.Annotate(
				NewEchoHandler,
				fx.As(new(Route)),
			),
			zap.NewExample,
		),
		fx.Invoke(func(*http.Server) {}),
	).Run()
}

// 我们引入了一个接口，将实现与消费者分离开来。
// 然后，我们用 fx.Annotate 和 fx.As 对先前提供的构造函数进行注解，将其结果转换为该接口。
// 这样，NewEchoHandler 就能继续返回 *EchoHandler。

// curl -X POST -d "你好，这是一个测试！" http://localhost:8080/echo
