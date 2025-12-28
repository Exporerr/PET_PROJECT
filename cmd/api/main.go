package main

import (
	"context"

	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	client "github.com/Explorerr/pet_project/internal/api-service/Client"
	middleware "github.com/Explorerr/pet_project/internal/api-service/Middleware"
	serviceapi "github.com/Explorerr/pet_project/internal/api-service/Service_api"

	handlerapi "github.com/Explorerr/pet_project/internal/api-service/Handler_api"

	kafkalogger "github.com/Explorerr/pet_project/pkg/Kafka_logger"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load("../../internal/api-service/configs/.env")
	if err != nil {
		log.Fatalf("не удалось загрузить конфиг: %v", err)

	}
	os.Mkdir("logs", 0755)
	f, err := os.OpenFile("logs/app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("не удалось открыть лог-файл: %v", err)
	}
	defer f.Close()
	var wg sync.WaitGroup
	var mu sync.Mutex
	ctx := context.Background()
	log := kafkalogger.New_Logger(f, 100, "API-SERVICE", &wg, &mu, time.Second*5)
	baseURL := os.Getenv("DB_SERVICE_URL")
	if baseURL == "" {
		log.ERROR("Api-service", "url-getting", "url env not set", nil)
		return

	}

	cli := client.NewClient(baseURL, *log)
	s := serviceapi.NewService_api(*cli, *log)

	h := handlerapi.New_Handler_api(*log, *s)
	r := mux.NewRouter()
	r.HandleFunc("/tasks/register", h.Create_New_user).Methods("POST")
	r.HandleFunc("/tasks/login", h.Login).Methods("POST")
	r.Handle("/tasks/create", middleware.JWT_Middleware(http.HandlerFunc(h.Create_Task), log)).Methods("POST")
	r.Handle("/tasks/up/{task-id}", middleware.JWT_Middleware(http.HandlerFunc(h.Update), log)).Methods("PATCH")
	r.Handle("/tasks/del/{task-id}", middleware.JWT_Middleware(http.HandlerFunc(h.Delete_Task), log)).Methods("DELETE")
	r.Handle("/tasks/get", middleware.JWT_Middleware(http.HandlerFunc(h.GetAllTasks), log)).Methods("GET")

	srv := http.Server{
		Addr:    ":8080",
		Handler: r,
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.INFO("Server", "Server starting", "Server started on 8080 port !", nil)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.ERROR("Server", "Server starting", "Server starting failed", nil)

		}

	}()
	<-stop
	log.INFO("Server", "Server shuting down", "Shutting down server...", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {

	}

}
