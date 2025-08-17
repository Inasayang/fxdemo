package main

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"go.uber.org/fx"
)

// 使用 fx.Lifecycle 对象为应用程序添加一个生命周期钩子。这将告诉 Fx 如何启动和停止 HTTP 服务器。
func NewHTTPServer(lc fx.Lifecycle) *http.Server {
	srv := &http.Server{Addr: ":8080"}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			fmt.Println("Starting HTTP server at", srv.Addr)
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
		// 使用 fx.Provide 为应用程序添加了一个 HTTP 服务器。该服务器与 Fx 应用程序生命周期挂钩--当我们调用 App.Run 时，它将开始为请求提供服务；当应用程序收到停止信号时，它将停止运行。
		// 我们使用 fx.Invoke 来请求始终实例化 HTTP 服务器，即使应用程序中没有其他组件直接引用它。
		fx.Provide(NewHTTPServer),        
		fx.Invoke(func(*http.Server) {}), 
	).Run()
}

// 问：fx.Invoke(func(*http.Server) {}) 是什么意思？
// 答：这一行的作用就是强制 Fx 框架去创建 *http.Server 实例，从而确保其生命周期钩子（OnStart 和 OnStop）被注册和执行。
// 1. fx.Provide 是懒加载的 (Lazy)
// 当你使用 fx.Provide(NewHTTPServer) 时，你只是在告诉 Fx 框架：“我这里有一个 NewHTTPServer 函数，如果你需要 *http.Server 类型的实例，就调用这个函数来创建它。”

// 关键点在于：Fx 默认是懒加载的。如果应用程序中没有其他任何部分明确表示需要一个 *http.Server，那么 Fx 就会认为没有必要去调用 NewHTTPServer 函数，这个 *http.Server 实例也就永远不会被创建。

// 2. fx.Invoke 的作用：强制执行
// fx.Invoke 的作用就是解决这个问题。它会注册一个函数，并告诉 Fx：“请立即执行这个函数，并为它提供所有它需要的依赖项。”
// 我们来看 fx.Invoke(func(*http.Server) {}) 这行代码：

// 它注册了一个匿名函数 func(*http.Server) {}。
// 这个函数的签名 func(*http.Server) 告诉 Fx，为了执行它，必须先提供一个 *http.Server 类型的参数。
// 为了满足这个依赖，Fx 就会去寻找 *http.Server 的提供者（Provider），找到了我们之前定义的 NewHTTPServer。
// 于是，Fx 调用 NewHTTPServer() 来创建一个 *http.Server 实例。
// NewHTTPServer() 被调用后，它内部的 lc.Append 逻辑就被成功执行，服务器的 OnStart 和 OnStop 钩子就被注册到 Fx 的生命周期中了。
// 最后，Fx 把创建好的 *http.Server 实例传给 Invoke 中的匿名函数。这个函数体是空的 {}，因为它不需要对这个实例做任何操作，它的唯一目的就是触发实例的创建。

// 3. 总结
// 你可以把 fx.Invoke 看作是应用程序的“入口”或“启动器”。它用来触发那些“只做事，不产出值给其他组件用”的逻辑，比如：

// 启动一个后台服务（就像这个例子中的 HTTP 服务器）
// 连接到数据库
// 注册事件监听器
// 所以，fx.Invoke(func(*http.Server) {}) 是一个在 Fx 中非常常见的模式，专门用来确保一个服务被初始化并启动，即使没有其他组件直接依赖它。
