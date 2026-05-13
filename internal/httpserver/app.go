package httpserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/MSNZT/orderflow/internal/config"
	"github.com/MSNZT/orderflow/internal/router"
)

type App struct {
	server *http.Server
	config *config.Config
}

func New(config *config.Config) *App {
	router := router.NewRouter()

	return &App{
		server: &http.Server{
			Addr:         config.Addr,
			Handler:      router,
			ReadTimeout:  config.Timeout,
			WriteTimeout: config.Timeout,
			IdleTimeout:  config.IdleTimeout,
		},
		config: config,
	}
}

func (a *App) Run(ctx context.Context) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		fmt.Printf("Starting server on %s\n", a.server.Addr)

		if err := a.server.ListenAndServe(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-stop

	fmt.Println("Получен сигнал завершения. Остановка сервера...")
	ctx, cancel := context.WithTimeout(ctx, a.config.ShutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		log.Fatalf("Произошла ошибка при остановке сервера")
	}

	fmt.Println("Сервер успешно остановлен")
}
