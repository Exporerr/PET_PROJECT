package handlerapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	serviceapi "github.com/Explorerr/pet_project/internal/api-service/Service_api"

	kafkalogger "github.com/Explorerr/pet_project/pkg/Kafka_logger"
	models "github.com/Explorerr/pet_project/pkg/Models"
	apperrors "github.com/Explorerr/pet_project/pkg/app_errors"
	contextkeys "github.com/Explorerr/pet_project/pkg/context_Key"
	"github.com/Explorerr/pet_project/pkg/kafkainit"
	"github.com/gorilla/mux"
)

type Handler struct {
	log      kafkalogger.ZapAdapter
	s        serviceapi.Service
	producer *kafkainit.Producer_real
}

func New_Handler_api(log kafkalogger.ZapAdapter, s serviceapi.Service, producer *kafkainit.Producer_real) *Handler {
	return &Handler{log: log, s: s}
}

func (h *Handler) Create_New_user(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var user models.Request_Register
	ctx := r.Context()
	if r.Header.Get("Content-Type") != "application/json" {
		h.log.ERROR("Hadler(api-service)", "Register-User", "Контент!=JSON", nil)
		http.Error(w, "Content-Type должен быть application/json", http.StatusUnsupportedMediaType)
		return
	}
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		h.log.ERROR("Handler(api-service)", "Register-User", fmt.Sprintf("Ошибка декодирования JSON: %v ", err), nil)

		http.Error(w, "неверный формат JSON", http.StatusBadRequest)
		return
	}
	h.log.DEBUG("Handler(api-service)", "Register", fmt.Sprintf("Post_Handler received user: %s", user.Email), nil)

	erro := h.s.Register(user, ctx)
	if erro != nil {
		h.log.ERROR("Hndler(api-service)", "Register", fmt.Sprintf("Пользовваель уже существует:%+v", erro), nil)

		http.Error(w, "Такой пользователь уже сущетвует ", http.StatusConflict)
		return
	}
	h.log.INFO("Handler(api-service)", "Register", " POST Handler успешно звершился", nil)

	w.WriteHeader(http.StatusCreated)

}
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var user models.Request_Login
	ctx := r.Context()
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		h.log.ERROR("Handler(api-service)", "Login", "Неверный Content-Type(Нужен json)", nil)
		http.Error(w, "Content-Type должен быть application/json", http.StatusUnsupportedMediaType)
		return

	}
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		h.log.ERROR("Handler(api-service)", "Login", fmt.Sprintf("Ошибка декодирования JSON: %V ", err), nil)

		http.Error(w, "неверный формат JSON", http.StatusBadRequest)
		return
	}
	h.log.DEBUG("Handler(api-service)", "Login", fmt.Sprintf("Post_Handler received user: %s", user.Email), nil)
	token, useer, erro := h.s.LoginService(user, ctx)
	if erro != nil {
		if errors.Is(erro, apperrors.ErrUserNotExist) {
			h.log.ERROR("Handler(api-service)", "Login", fmt.Sprintf("Пользовваель не существует:%+v", erro), nil)
			http.Error(w, "Такой пользователь не сущетвует ", http.StatusNotFound)
			return

		}
		h.log.ERROR("Handler(api-service)", "Login", fmt.Sprintf("internal server error:%+v", erro), nil)

		http.Error(w, "internal error", http.StatusInternalServerError)
		return

	}
	h.log.INFO("Handler(api-service)", "Login", " Login Handler пользователь успешно авторизовался ", nil)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := map[string]interface{}{
		"message": "Успешная авторизация",
		"token":   token,
		"user": map[string]interface{}{
			"name":  useer.Username,
			"email": useer.Email,
			"role":  useer.Role,
		},
	}
	go h.producer.WriteMessagee(useer.ID, "Login", 0, r)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.log.ERROR("Handler(api-service)", "Login(json encoding)", fmt.Sprintf("Ошибка при кодитровании json:  %v", err), &useer.ID)
		http.Error(w, "Ошибка кодирования json ", http.StatusBadRequest)
		return
	}
	h.log.INFO("Handler(api-service", "Login", "Пользователь успешно вошел", &useer.ID)

}

func (h *Handler) Create_Task(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	ctx := r.Context()
	userID, ok := r.Context().Value(contextkeys.UserID).(int)
	if !ok {
		h.log.ERROR("Handler(api-service)", "Create_Task", "id отсутсвтует в контексте", &userID)
		http.Error(w, "Неавторизован", http.StatusUnauthorized)
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		h.log.ERROR("Handler(api-service)", "Create_Task", "Неверный Content-Type(Нужен json)", &userID)
		http.Error(w, "Content-Type должен быть application/json", http.StatusUnsupportedMediaType)
		return
	}

	var task models.Request_Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		h.log.ERROR("Handler(api-service)", "Create_task", "Ошибка декодирование JSON ", &userID)
		http.Error(w, "Ошибка декодирования JSON тела", http.StatusBadRequest)
		return
	}

	Task, err := h.s.Create_Task(userID, task, ctx)
	if err != nil {
		h.log.ERROR("Handler(api-service)", "Create_Task", "Ошибка при создании задачи", &userID)
		http.Error(w, "Ошибка при создании задачи", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
	resp := map[string]interface{}{
		"message": "Задача успешно создана",

		"task": map[string]interface{}{
			"id":          Task.ID,
			"title":       Task.Title,
			"description": Task.Description,
			"status":      Task.Status,
			"created_at":  Task.CreatedAt,
		},
	}
	go h.producer.WriteMessagee(userID, "Create-Task", Task.ID, r)

	erro := json.NewEncoder(w).Encode(resp)
	if erro != nil {
		h.log.ERROR("Handler(api-service)", "Create_Task(json encoding)", fmt.Sprintf("Ошибка при кодитровании json:  %v", erro), &userID)
		http.Error(w, "Ошибка кодирования json ", http.StatusBadRequest)
		return
	}
	h.log.INFO("Handler(api-service)", "Create_Task", "Задача успешно создана", &userID)

}

func (h *Handler) Delete_Task(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	ctx := r.Context()
	vars := mux.Vars(r)
	userID, ok := r.Context().Value(contextkeys.UserID).(int)
	if !ok {
		h.log.ERROR("Handler(api-service)", "Delete_Task", "id отсутсвтует в контексте", nil)
		http.Error(w, "Неавторизован", http.StatusUnauthorized)
		return
	}
	taskStr := vars["task-id"]

	new_taskID, err := strconv.Atoi(taskStr)
	if err != nil {
		h.log.ERROR("Handler(api-service)", "Delete_Task", fmt.Sprintf("Неверный формат task_id: %s", taskStr), &userID)
		http.Error(w, "Invalid task ID format", http.StatusBadRequest)
		return
	}

	erro := h.s.Delete(ctx, userID, new_taskID)
	if erro != nil {
		if errors.Is(erro, apperrors.ErrTaskNotFound) {
			h.log.ERROR("Handler(api-service)", "Delete_Task", "Задача не найдена", &userID)
			http.Error(w, "Задача не найдена", http.StatusNotFound)
			return

		}
		h.log.ERROR("Handler(api-service)", "Delete_Task", "Неизвестная ошибка либо упала транзакция", &userID)
		http.Error(w, "Ошибка", http.StatusInternalServerError)
		return
	}
	go h.producer.WriteMessagee(userID, "Create-Task", new_taskID, r)
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Task with id|%d| deleted succesfully", new_taskID),
	}); err != nil {
		h.log.ERROR("Handler(api-service)", "Delete_Task", fmt.Sprintf("Ошибка кодирования json: %v", err), &userID)
		return
	}

	h.log.INFO("Handler(api-service)", "Delete_Task", "Задача успешно удалена", &userID)

}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	ctx := r.Context()
	vars := mux.Vars(r)
	userID, ok := r.Context().Value(contextkeys.UserID).(int)
	if !ok {
		h.log.ERROR("Handler(api-service)", " Update", "id отсутсвтует в контексте", nil)
		http.Error(w, "Неавторизован", http.StatusUnauthorized)
		return
	}
	taskStr := vars["task-id"]

	new_taskID, err := strconv.Atoi(taskStr)
	if err != nil {
		h.log.ERROR("Handler(api-service)", "Delete_Task", fmt.Sprintf("Неверный формат task_id: %s", taskStr), &userID)
		http.Error(w, "Invalid task ID format", http.StatusBadRequest)
		return
	}

	erro := h.s.Update(ctx, userID, new_taskID)
	if erro != nil {
		if errors.Is(erro, apperrors.ErrTaskNotFound) {
			h.log.ERROR("Handler(api-service)", "Update", "Задача не найдена", &userID)
			http.Error(w, "Задача не найдена", http.StatusNotFound)
			return

		}
		h.log.ERROR("Handler(api-service)", "Update", "Неизвестная ошибка либо упала транзакция", &userID)
		http.Error(w, "Ошибка", http.StatusInternalServerError)
		return

	}
	go h.producer.WriteMessagee(userID, "Create-Task", new_taskID, r)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Task with id|%d| done(updated) succesfully", new_taskID),
	}); err != nil {
		h.log.ERROR("Handler(api-service)", "Update", fmt.Sprintf("Ошибка кодирования json: %v", err), &userID)
		return
	}

	h.log.INFO("Handler(api-service)", "Update", "Задача успешно  выполнена ", &userID)

}

func (h *Handler) GetAllTasks(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	ctx := r.Context()
	userID, ok := r.Context().Value(contextkeys.UserID).(int)
	if !ok {
		h.log.ERROR("Handler(api-service)", "GetAllTasks", "id отсутсвтует в контексте", nil)
		http.Error(w, "Неавторизован", http.StatusUnauthorized)
		return
	}
	task, err := h.s.GetAllTasks(userID, ctx)
	if err != nil {
		if errors.Is(err, apperrors.ErrEmptySlice) {
			h.log.INFO("Handler(api-service)", "GetAllTasks", "Нету задач", &userID)
			http.Error(w, "У вас неу задач", http.StatusNotFound)
			return
		}
		h.log.ERROR("Handler(api-service)", "GetAllTasks", fmt.Sprintf("непонятная ошибка %v", err), &userID)
		http.Error(w, "Ошибка при получении задач", http.StatusInternalServerError)
		return
	}
	go h.producer.WriteMessagee(userID, "Create-Task", 0, r)
	w.WriteHeader(http.StatusOK)
	erro := json.NewEncoder(w).Encode(task)
	if erro != nil {
		h.log.ERROR("Handler(api-service)", "Json encoding", "Ошибка при кодировании Task в Json", &userID)
		http.Error(w, "Ошибка кодирования json ", http.StatusBadRequest)
		return
	}

}
