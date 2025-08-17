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

// 在 EchoHandler 中添加一个字段来保存日志记录器，并在 NewEchoHandler 中添加一个新的日志记录器参数来设置该字段。
type EchoHandler struct {
	log *zap.Logger
}

func NewEchoHandler(log *zap.Logger) *EchoHandler {
	return &EchoHandler{log: log}
}

// 在 EchoHandler.ServeHTTP 方法中，使用日志记录器而不是打印到标准错误。
func (h *EchoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, err := io.Copy(w, r.Body); err != nil {
		h.log.Warn("Failed to handle request", zap.Error(err))
	}else {
		h.log.Info("Request handled successfully", zap.String("method", r.Method), zap.String("url", r.URL.String()))
	}
}

func NewServeMux(echo *EchoHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/echo", echo)
	return mux
}

// 同样，更新 NewHTTPServer 以期待一个日志记录器，并将 "Starting HTTP server（开始 HTTP 服务器）"消息记录到日志记录器中。
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
		// 您也可以使用同一个 Zap 日志记录器来记录 Fx 自己的日志。
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
		fx.Provide(
			NewHTTPServer,
			// 请注意，NewServeMux 被添加到 NewEchoHandler 的上方--向 fx.Provide 提供构造函数的顺序并不重要。
			NewServeMux, 
			NewEchoHandler,
			// 为应用程序提供一个 Zap 日志记录器。
			// 在本教程中，我们将使用 zap.NewExample。
			// 但在实际应用中，你应该使用 zap.NewProduction 或创建一个更个性化的日志记录器。
			zap.NewExample, 
		),
		fx.Invoke(func(*http.Server) {}),
	).Run()
}

// curl -X POST -d "你好，这是一个测试！" http://localhost:8080/echo
