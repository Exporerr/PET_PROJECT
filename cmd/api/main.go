package main

import (
	"context"
	"fmt"

	"log"
	"net/http"
	"os"
	"os/signal"

	"syscall"
	"time"

	client "github.com/Explorerr/pet_project/internal/api-service/Client"
	middleware "github.com/Explorerr/pet_project/internal/api-service/Middleware"
	serviceapi "github.com/Explorerr/pet_project/internal/api-service/Service_api"

	handlerapi "github.com/Explorerr/pet_project/internal/api-service/Handler_api"

	kafkalogger "github.com/Explorerr/pet_project/pkg/Kafka_logger"
	"github.com/Explorerr/pet_project/pkg/kafkainit"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Создание контекста
	ctx, cancel := context.WithCancel(context.Background())
	err := godotenv.Load("../../internal/api-service/configs/.env")
	if err != nil {
		log.Fatalf("не удалось загрузить конфиг: %v", err)

	}

	log, logfileClose, err := kafkalogger.New_ZapAdapter("API-SERVICE", "DEBUG")
	if err != nil {
		panic(err)
	}
	defer log.Logger.Sync()
	defer logfileClose()
	// Создание Брокера конфига
	err, conn := kafkainit.Set_Config()
	if err != nil {
		log.ERROR("Server", "Creating conn and Kafka config", "failed to create conn and config", nil)
		return
	}
	// Создание Консьюмера
	consumer, err, event_file_close := kafkainit.New_Consumer(log, ctx)

	if err != nil {
		log.ERROR("Server", "Creating Consummer", "failed to create consumer and log file", nil)
		return
	}
	go consumer.Run(ctx)

	// Создание Продюссера
	producer := kafkainit.New_Producer(log, 4, 5)
	go producer.Run(ctx)

	// Достаем адресс db-service из env
	baseURL := os.Getenv("DB_SERVICE_URL")
	if baseURL == "" {
		log.ERROR("Api-service", "url-getting", "url env not set", nil)
		return

	}
	// Создание Клиента для запросов к db-service(у)
	cli := client.NewClient(baseURL, *&log)
	// Создание Сервиса
	s := serviceapi.NewService_api(*cli, *log)
	// Создание Хедлера
	h := handlerapi.New_Handler_api(*log, *s, producer)
	r := mux.NewRouter()
	r.HandleFunc("/tasks/register", h.Create_New_user).Methods("POST")
	r.HandleFunc("/tasks/login", h.Login).Methods("POST")
	r.Handle("/tasks/create", middleware.Info_Middleware(middleware.JWT_Middleware(http.HandlerFunc(h.Create_Task), log), log)).Methods("POST")
	r.Handle("/tasks/up/{task-id}", middleware.Info_Middleware(middleware.JWT_Middleware(http.HandlerFunc(h.Update), log), log)).Methods("PATCH")
	r.Handle("/tasks/del/{task-id}", middleware.Info_Middleware(middleware.JWT_Middleware(http.HandlerFunc(h.Delete_Task), log), log)).Methods("DELETE")
	r.Handle("/tasks/get", middleware.Info_Middleware(middleware.JWT_Middleware(http.HandlerFunc(h.GetAllTasks), log), log)).Methods("GET")

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
	shutdown_ctx, shut_cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*15))
	defer shut_cancel()
	if err := srv.Shutdown(shutdown_ctx); err != nil {
		log.ERROR("Server", "Shutdown", fmt.Sprintf("Shutdown error: %v", err), nil)

	}
	cancel()           // 1. отменяем appCtx
	consumer.Close()   // 2. ждём завершения consumer
	producer.Close()   // 3. закрываем producer
	event_file_close() // 4. закрываем файл consumer
	conn.Close()       // 5. Kafka infra
	logfileClose()     // 6. лог файл
	log.Logger.Sync()

}
