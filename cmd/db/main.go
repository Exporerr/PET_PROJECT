package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"log"
	"net/http"
	"os"
	"sync"
	"time"

	handler "github.com/Explorerr/pet_project/internal/db-service/Handler"
	repository "github.com/Explorerr/pet_project/internal/db-service/Repository"
	service "github.com/Explorerr/pet_project/internal/db-service/Service"
	serviceoriginal "github.com/Explorerr/pet_project/internal/db-service/service_original"
	kafkalogger "github.com/Explorerr/pet_project/pkg/Kafka_logger"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load("../../internal/db-service/configs/.env")
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
	log := kafkalogger.New_Logger(f, 100, "DB-SERVICE", &wg, &mu, time.Second*5)
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	if host == "" || port == "" || user == "" || password == "" || dbname == "" {
		log.ERROR("Error miss the string of database", "", "", nil)
		fmt.Println("Пустой ENV")
		return
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	pool, err := repository.NewPool(ctx, dsn)

	if err != nil {
		log.ERROR("Server", "Server starting", "Pool connection failed", nil)

	}
	defer pool.Close()
	db_conn := repository.NewUserPool(pool, *log)
	repo := repository.NewRepo(db_conn)
	s := service.New_Service(repo, log)
	serv := serviceoriginal.New_User_service(s)
	h := handler.NewHandler(log, serv)
	r := mux.NewRouter()
	r.HandleFunc("/tasks/register", h.POST_USER).Methods("POST")
	r.HandleFunc("/tasks/login", h.Login).Methods("POST")
	r.HandleFunc("/tasks", h.Create_Task).Methods("POST")
	r.HandleFunc("/tasks/up/{user_id}/{tasl-id}", h.Update).Methods("PATCH")
	r.HandleFunc("/tasks/del/{user_id}/{tasl-id}", h.Delete_Task).Methods("DELETE")
	r.HandleFunc("/tasks/{id}", h.GetAllTasks).Methods("GET")

	srv := http.Server{
		Addr:    ":8081",
		Handler: r,
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.INFO("Server", "Server starting", "Server started on 8081 port !", nil)
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
