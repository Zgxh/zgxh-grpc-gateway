package gateway

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang/glog"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type Options struct {
	Mux []gwruntime.ServeMuxOption
}

func StartGateway() {
	// 启动一个 context
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 与grpc server之间建立一个 tcp 连接
	conn, err := dialTcp(ctx, GrpcServerAddr)
	if err != nil {
		glog.Errorf("Failed to build the tcp connection: %v", err)
	}

	// 起一个协程：当任务执行完毕后，关闭连接 conn
	go func() {
		<-ctx.Done()
		if err := conn.Close(); err != nil {
			glog.Errorf("Failed to close a client connection to the gRPC server: %v", err)
		}
	}()

	// 启动一个http服务器
	mux := http.NewServeMux()

	opts := Options{}

	// 启动一个gateway服务器
	gwMux := gwruntime.NewServeMux(opts.Mux...)
	// 注册grpc api
	for _, f := range Apis {
		if err := f(ctx, gwMux, conn); err != nil {
			glog.Errorf("Failed to register the grpc apis: %v", err)
		}
	}

	mux.Handle("/", gwMux)

	s := &http.Server{
		Addr:    Addr,
		Handler: allowCORS(mux),
	}
	go func() {
		<-ctx.Done()
		glog.Infof("Shutting down the http server")
		if err := s.Shutdown(context.Background()); err != nil {
			glog.Errorf("Failed to shutdown http server: %v", err)
		}
	}()

	// 监听8080端口，对外提供http服务
	glog.Infof("Starting listening at %s", Addr)
	if err := s.ListenAndServe(); err != http.ErrServerClosed {
		glog.Errorf("Failed to listen and serve: %v", err)
	}
}

func allowCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			if r.Method == "OPTIONS" && r.Header.Get("Access-Control-Request-Method") != "" {
				preflightHandler(w, r)
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}

// 建立 tcp 连接
func dialTcp(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	return grpc.DialContext(ctx, addr, grpc.WithInsecure())
}

func preflightHandler(w http.ResponseWriter, r *http.Request) {
	headers := []string{"Content-Type", "Accept", "Authorization"}
	w.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ","))
	methods := []string{"GET", "HEAD", "POST", "PUT", "DELETE"}
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ","))
	glog.Infof("preflight request for %s", r.URL.Path)
}
