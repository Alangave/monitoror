package service

import (
	"fmt"
	"time"

	"github.com/monitoror/monitoror/cli/debug"
	"github.com/monitoror/monitoror/service/handlers"
	"github.com/monitoror/monitoror/service/middlewares"
	"github.com/monitoror/monitoror/store"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/color"
)

type (
	Server struct {
		// Echo Server
		*echo.Echo

		// CacheMiddleware using CacheStore to return cached data
		CacheMiddleware *middlewares.CacheMiddleware

		store *store.Store
	}
)

var colorer = color.New()

// Init create echo server with middlewares, ui, routes
func Init(store *store.Store) *Server {
	s := &Server{
		store: store,
	}

	s.setupEchoServer()
	s.setupEchoMiddleware()

	InitUI(s)
	InitApis(s)

	return s
}

func (s *Server) Start() error {
	return s.Echo.Start(fmt.Sprintf("%s:%d", s.store.CoreConfig.Address, s.store.CoreConfig.Port))
}

func (s *Server) setupEchoServer() {
	s.Echo = echo.New()
	s.HideBanner = true
	s.HidePort = true

	// ----- Errors Handler -----
	s.HTTPErrorHandler = handlers.HTTPErrorHandler
}

func (s *Server) setupEchoMiddleware() {
	// Recover (don't panic 😎)
	s.Use(echoMiddleware.Recover())

	// Log requests
	if debug.IsEnabled() {
		s.Use(echoMiddleware.RequestLoggerWithConfig(echoMiddleware.RequestLoggerConfig{
			LogMethod:  true,
			LogURI:     true,
			LogStatus:  true,
			LogLatency: true,
			LogError:   true,
			LogValuesFunc: func(c echo.Context, v echoMiddleware.RequestLoggerValues) error {
				errorMessage := ""
				if v.Error != nil {
					errorMessage = v.Error.Error()
				}

				fmt.Printf("[-] %s %s status:%d latency:%s error:%q\n",
					colorer.Green(v.Method), v.URI, v.Status, colorer.Green(v.Latency.String()), errorMessage,
				)
				return nil
			},
		}))
	}

	// Cache
	s.CacheMiddleware = middlewares.NewCacheMiddleware(s.store.CacheStore,
		time.Millisecond*time.Duration(s.store.CoreConfig.DownstreamCacheExpiration),
		time.Millisecond*time.Duration(s.store.CoreConfig.UpstreamCacheExpiration),
	) // Used as Handler wrapper in routes
	s.Use(s.CacheMiddleware.DownstreamStoreMiddleware())

	// CORS
	s.Use(echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.POST},
	}))
}
