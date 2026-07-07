package main

import (
	"context"
	"errors"
	"fmt"
	"gateway/internal/auditor"
	"gateway/internal/config"
	"gateway/internal/controller"
	"gateway/internal/proxy"
	ratelimit "gateway/internal/rate-limit"
	"gateway/internal/request"
	"gateway/internal/services"
	"gateway/internal/store"
	"gateway/internal/telemeter"
	"gateway/internal/validate"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/rs/cors"
)

func main() {
	args := os.Args
	fmt.Print(args[1])
	if len(args) > 0 && args[1] == "read" {
		readInternal("./internal")
		return
	}

	var configLoader config.Loader = config.NewConfigLoader("./config.json")
	c := configLoader.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	scm := c.GetServiceConfigMap()
	var validator validate.Validator = validate.NewReqValidator(c.BlockedIps, c.AllowedOrigins, c.ReqSizeLimitInBytes, scm)
	var proxier proxy.Proxier = proxy.ReqProxy{}
	var rateLimiter ratelimit.RateLimiter = ratelimit.NewReqRateLimiter(ctx, c.RateLimitPerMinute, scm)
	var serviceAccessController controller.ServiceAccessController = controller.ReqServiceAccessController{}
	var requestTelemeter telemeter.Recorder = telemeter.NewReqTelemeter(scm)
	var storer store.Storer = store.NewStorage(ctx)
	corsPolicy := cors.New(cors.Options{
		AllowedOrigins: append([]string{}, c.AllowedOrigins...),
	})

	handler, err := request.NewHandler(request.HandleRequestData{
		Config:                  c,
		Validator:               validator,
		RateLimiter:             rateLimiter,
		ServiceAccessController: serviceAccessController,
		Proxier:                 proxier,
		Telemter:                requestTelemeter,
		GetService: func(scs []config.ServiceConfig) services.Storer {
			return services.NewMockServiceStore(scs)
		},
	})

	if err != nil {
		panic(err)
	}

	handler = corsPolicy.Handler(handler)
	handler = auditor.NewHandler(ctx, requestTelemeter, storer, handler)

	mux := http.NewServeMux()
	mux.Handle("GET /", handler)
	mux.Handle("GET /logs", storer.ReadLogsHandler())

	server := &http.Server{
		ReadHeaderTimeout: c.GlobalTimeoutInSeconds,
		ReadTimeout:       c.GlobalTimeoutInSeconds,
		WriteTimeout:      c.GlobalTimeoutInSeconds,
		Handler:           mux,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed to close the server: %s", err.Error())
		}
		log.Fatalf("stopped server new connections")
	}()

	<-ctx.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 2*c.GlobalTimeoutInSeconds)
	defer cancel()

	if err = server.Shutdown(ctx); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("failed to shutdown the server: %s", err.Error())
	}
}

func readInternal(path string) {
	dir, err := os.ReadDir(path)
	if err != nil {
		panic(err)
	}

	for _, file := range dir {
		fileName := file.Name()
		fullPath := filepath.Join(path, fileName)
		if !file.IsDir() {
			data, err := os.ReadFile(fullPath)
			if err != nil {
				panic(err)
			}
			fmt.Println("---logging file---", fullPath)
			fmt.Print(string(data))
		} else {
			readInternal(fullPath)
		}

	}
}
