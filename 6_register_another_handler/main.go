package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

type Route interface {
	http.Handler
	Pattern() string
}

type EchoHandler struct {
	log *zap.Logger
}

func NewEchoHandler(log *zap.Logger) *EchoHandler {
	return &EchoHandler{log: log}
}

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

// HelloHandler is an HTTP handler that
// prints a greeting to the user.
type HelloHandler struct {
	log *zap.Logger
}

// NewHelloHandler builds a new HelloHandler.
func NewHelloHandler(log *zap.Logger) *HelloHandler {
	return &HelloHandler{log: log}
}

func (*HelloHandler) Pattern() string {
	return "/hello"
}

func (h *HelloHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.log.Error("Failed to read request", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if _, err := fmt.Fprintf(w, "Hello, %s\n", body); err != nil {
		h.log.Error("Failed to write response", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func NewServeMux(route1, route2 Route) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle(route1.Pattern(), route1)
	mux.Handle(route2.Pattern(), route2)
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
			// 在 main() 中注释 NewServeMux，以选取这两个名称值。
			fx.Annotate(
				NewServeMux,
				// 关键：告诉 Fx，第一个 Route 参数要找名字叫 "echo" 的，第二个要找名字叫 "hello" 的
				fx.ParamTags(`name:"echo"`, `name:"hello"`),
			),
			fx.Annotate(
				NewEchoHandler,
				fx.As(new(Route)),
				// 关键：给这个由 NewEchoHandler 返回的 Route 实例起个名字，叫做 "echo"
				fx.ResultTags(`name:"echo"`), 
			),
			fx.Annotate(
				NewHelloHandler,
				fx.As(new(Route)),
				// 关键：给这个由 NewHelloHandler 返回的 Route 实例起个名字，叫做 "hello"
				fx.ResultTags(`name:"hello"`),
			),
			zap.NewExample,
		),
		fx.Invoke(func(*http.Server) {}),
	).Run()
}

// 我们添加了一个构造函数，用于生成与现有类型相同的值。
// 我们用 fx.ResultTags 对构造函数进行了注解，以产生已命名的值，并用 fx.ParamTags 对消费者进行了注解，以消费这些已命名的值。

// curl -X POST -d "你好，这是一个测试！" http://localhost:8080/echo
// curl -X POST -d "你好，这是一个测试！" http://localhost:8080/hello
