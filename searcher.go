package main

import (
	"context"
	"fmt"
	"github.com/cortezaproject/corteza-discovery-searcher/searcher"
	"github.com/cortezaproject/corteza-server/pkg/cli"
	"github.com/cortezaproject/corteza-server/pkg/logger"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/jwtauth"
	"go.uber.org/zap"
	"net"
	"net/http"
)

func main() {
	cfg, err := getConfig()
	cli.HandleError(err)

	log := logger.MakeDebugLogger().WithOptions(zap.AddStacktrace(zap.PanicLevel))
	ctx := cli.Context()

	api, err := searcher.ApiClient(cfg.cortezaHttp, cfg.cortezaAuth, cfg.clientKey, cfg.clientSecret)
	cli.HandleError(err)

	esc, err := searcher.EsClient(cfg.es.addresses)
	cli.HandleError(err)

	StartHttpServer(ctx, log, cfg.httpAddr, func() http.Handler {
		router := chi.NewRouter()
		router.Use(handleCORS)
		router.Use(middleware.StripSlashes)
		router.Use(middleware.RealIP)
		router.Use(middleware.RequestID)

		if len(cfg.jwtSecret) == 0 {
			log.Warn(fmt.Sprintf("JWT secret not set (%s), access to private indexes disabled", envKeyJwtSecret))
		} else {
			router.Use(jwtauth.Verifier(jwtauth.New(jwt.SigningMethodHS512.Alg(), cfg.jwtSecret, nil)))
		}

		// @todo If we want to prevent any kind of anonymous access
		//router.Use(jwtauth.Authenticator)

		searcher.Handlers(router, log, esc, api)

		return router
	}())
}

func StartHttpServer(ctx context.Context, log *zap.Logger, addr string, h http.Handler) {

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error("cannot start server", zap.Error(err))
		return
	}

	go func() {
		srv := http.Server{
			Handler: h,
			BaseContext: func(listener net.Listener) context.Context {
				return ctx
			},
		}
		log.Info("http server started", zap.String("addr", addr))
		err = srv.Serve(listener)
	}()
	<-ctx.Done()
}

// Sets up default CORS rules to use as a middleware
func handleCORS(next http.Handler) http.Handler {
	return cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-ID"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}).Handler(next)
}
