package main

import (
	"astro/pkg/handler"
	repo "astro/pkg/repository"
	srvc "astro/pkg/service"
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetFormatter(new(logrus.JSONFormatter))

	cnf := repo.Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		Username: os.Getenv("DB_USERNAME"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
		Password: os.Getenv("DB_PASSWORD"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := repo.NewPostgresDB(ctx, cnf)
	if err != nil {
		logrus.Fatalf("failed to initialize db: %s", err.Error())
	}

	handlers := handler.NewHandler(srvc.NewService(repo.NewRepository(db)))

	srv := new(server)
	go func() {
		if err := srv.Run(os.Getenv("APP_PORT"), handlers.InitRoutes()); err != nil {
			logrus.Fatalf("error occured while running http server: %s", err.Error())
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	logrus.Printf("astro Shutting Down")

	if err := srv.Shutdown(ctx); err != nil {
		logrus.Errorf("error occured on server shutting down: %s", err.Error())
	}

	if err := db.Close(); err != nil {
		logrus.Errorf("error occured on db connection close: %s", err.Error())
	}
}

type server struct {
	httpSrv *http.Server
}

func (s *server) Run(port string, h http.Handler) error {
	s.httpSrv = &http.Server{
		Addr:           ":" + port,
		Handler:        h,
		MaxHeaderBytes: 1 << 20,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    10 * time.Second,
	}

	return s.httpSrv.ListenAndServe()
}

func (s *server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}
