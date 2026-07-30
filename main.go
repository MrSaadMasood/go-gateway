package main

import (
	"context"
	"errors"
	"fmt"
	"gateway/internal/auditor"
	"gateway/internal/config"
	"gateway/internal/gateway"
	glog "gateway/internal/log"
	"gateway/internal/proxy"
	ratelimit "gateway/internal/rate-limit"
	"gateway/internal/request"
	"gateway/internal/route"
	"gateway/internal/services"
	"gateway/internal/store"
	"gateway/internal/telemeter"
	"gateway/internal/validate"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func main() {

	args := os.Args
	fmt.Print(args[1])
	if len(args) > 0 && args[1] == "read" {
		readInternal("./internal")
		return
	}

	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var configLoader config.Loader = config.NewConfigLoader("./config.json")
	storage := store.NewStorage(ctx)
	var logger glog.Logger = glog.New(ctx)
	var storer store.Storer = storage

	g := gateway.New(ctx, configLoader, storage, logger)
	c := g.Start()
	scm := c.GetServiceConfigMap()

	var validator validate.Validator = validate.NewReqValidator(c.BlockedIps, c.AllowedOrigins, c.ReqSizeLimitInBytes, scm)
	var proxier proxy.Proxier = proxy.NewReqProxy(c.GetGlobalTimeout())
	var rateLimiter ratelimit.RateLimiter = ratelimit.NewReqRateLimiter(ctx, c.RateLimitPerMinute, scm)
	var requestTelemeter telemeter.Recorder = telemeter.NewReqTelemeter(scm)
	var router route.Router = route.NewReqRouter()
	corsPolicy := cors.New(cors.Options{
		AllowedOrigins: append([]string{}, c.AllowedOrigins...),
	})

	reqAuditor := auditor.NewReqAuditor(ctx, storer, logger)

	handler, err := request.NewHandler(request.HandleRequestData{
		Config:      c,
		Validator:   validator,
		RateLimiter: rateLimiter,
		Proxier:     proxier,
		Telemter:    requestTelemeter,
		Router:      router,
		GetService: func(scs []config.ServiceConfig) services.Storer {
			return services.NewServiceStore(scs)
		},
	})

	if err != nil {
		panic(err)
	}

	handler = corsPolicy.Handler(handler)
	handler = auditor.NewHandler(reqAuditor, requestTelemeter, handler)

	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.Handle("GET /logs", storer.ReadLogsHandler())

	server := &http.Server{
		ReadHeaderTimeout: time.Duration(c.GetGlobalTimeout()),
		ReadTimeout:       time.Duration(c.GetGlobalTimeout()),
		WriteTimeout:      time.Duration(c.GetGlobalTimeout()),
		Handler:           mux,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			logger.Log(slog.LevelError, "SERVER_ERROR", slog.String("error", err.Error()))
		}
		logger.Log(slog.LevelError, "SERVER_STOPPED", slog.String("message", "server closed, stopped accepting new connections"))
	}()

	<-ctx.Done()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(2*c.GlobalTimeoutInSeconds))
	defer cancel()

	if err = server.Shutdown(ctx); !errors.Is(err, http.ErrServerClosed) {
		logger.Log(slog.LevelError, "SERVER_ERROR", slog.String("error", err.Error()))
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
