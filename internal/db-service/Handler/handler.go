package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	serviceoriginal "github.com/Explorerr/pet_project/internal/db-service/service_original"
	kafkalogger "github.com/Explorerr/pet_project/pkg/Kafka_logger"
	models "github.com/Explorerr/pet_project/pkg/Models"
	apperrors "github.com/Explorerr/pet_project/pkg/app_errors"
	"github.com/gorilla/mux"
)

type Handler struct {
	log kafkalogger.LoggerInterface
	s   serviceoriginal.User_Interface
}

func NewHandler(log kafkalogger.LoggerInterface, service serviceoriginal.User_Interface) *Handler {
	return &Handler{log: log, s: service}
}

func (h *Handler) POST_USER(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var user models.Request_Register
	ctx := r.Context()
	if r.Header.Get("Content-Type") != "application/json" {
		h.log.ERROR("Hnadler(db-service)", "Register-User", "Контент!=JSON", nil)
		http.Error(w, "Content-Type должен быть application/json", http.StatusUnsupportedMediaType)
		return
	}
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		h.log.ERROR("Handler(db-service)", "Register-User", fmt.Sprintf("Ошибка декодирования JSON: %v ", err), nil)

		http.Error(w, "неверный формат JSON", http.StatusBadRequest)
		return
	}
	h.log.DEBUG("Handler(db-service)", "Register", fmt.Sprintf("Post_Handler received user: %s", user.Email), nil)

	erro := h.s.Create_New_user(ctx, user)
	if erro != nil {
		h.log.ERROR("Handler(db-service", "Register", fmt.Sprintf("Пользователь уже существует:%+v", erro), nil)
		http.Error(w, "Такой пользователь уже существует ", http.StatusConflict)
		return
	}
	h.log.INFO("Handler(db-service)", "Register", " POST Handler успешно завершился", nil)

	w.WriteHeader(http.StatusCreated)

}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var user models.Request_Login
	ctx := r.Context()
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		h.log.ERROR("Handler(db-service)", "Login", "Неверный Content-Type(Нужен json)", nil)
		http.Error(w, "Content-Type должен быть application/json", http.StatusUnsupportedMediaType)
		return

	}
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		h.log.ERROR("Handler(db-service)", "Login", fmt.Sprintf("Ошибка декодирования JSON: %V ", err), nil)

		http.Error(w, "неверный формат JSON", http.StatusBadRequest)
		return
	}
	h.log.DEBUG("Handler(db-service)", "Login", fmt.Sprintf("Post_Handler received user: %s", user.Email), nil)
	useer, erro := h.s.Login(ctx, user.Email)
	if erro != nil {
		if errors.Is(erro, apperrors.ErrUserNotExist) {
			h.log.ERROR("Handler(db-service", "Login", fmt.Sprintf("Пользовваель не существует:%+v", erro), nil)
			http.Error(w, "Такой пользователь не сущетвует ", http.StatusNotFound)
			return

		}
		h.log.ERROR("Handler(db-service", "Login", fmt.Sprintf("internal server error:%+v", erro), nil)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return

	}
	h.log.INFO("Handler(db-service)", "Login", " Login Handler успешно звершился", nil)
	w.WriteHeader(http.StatusOK)
	erroo := json.NewEncoder(w).Encode(useer)
	if erroo != nil {
		h.log.ERROR("Handler", "Login(json encoding)", fmt.Sprintf("Ошибка при кодировании json:  %v", erroo), &useer.ID)
		http.Error(w, "Ошибка кодирования json ", http.StatusBadRequest)
		return

	}
	h.log.INFO("Handler(db-service)", "Login", "Пользователь успешно вошел", &useer.ID)

}

func (h *Handler) Create_Task(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	ctx := r.Context()
	userID := r.URL.Query().Get("user_id")
	user_newID, erroo := strconv.Atoi(userID)
	if erroo != nil {
		h.log.ERROR("Handler(db-service)", "Create_Task", "Ошибка чтения user_id", &user_newID)
		http.Error(w, "Неверный формат user-id ", http.StatusUnsupportedMediaType)
		return

	}

	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		h.log.ERROR("Handler(db-service)", "Create_Task", "Неверный Content-Type(Нужен json)", &user_newID)
		http.Error(w, "Content-Type должен быть application/json", http.StatusUnsupportedMediaType)
		return
	}
	var task models.Request_Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		h.log.ERROR("Handler(db-service)", "Create_task", "Ошибка декодирование JSON ", &user_newID)
		http.Error(w, "Ошибка декодирования JSON тела", http.StatusBadRequest)
		return
	}

	Task, err := h.s.Create_Task(ctx, &task, user_newID)
	if err != nil {
		h.log.ERROR("Handler(db-service)", "Create_Task", "Ошибка при создании задачи", &user_newID)
		http.Error(w, "Ошибка при создании задачи", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)

	erro := json.NewEncoder(w).Encode(Task)
	if erro != nil {
		h.log.ERROR("Handler", "Create_Task(json encoding)", fmt.Sprintf("Ошибка при кодитровании json:  %v", erro), &user_newID)
		http.Error(w, "Ошибка кодирования json ", http.StatusBadRequest)
		return

	}
	h.log.INFO("Handler(db-service)", "Create_Task", "Задача успешно создана", &user_newID)

}

func (h *Handler) Delete_Task(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	ctx := r.Context()

	vars := mux.Vars(r)
	idStr := vars["user_id"]
	taskStr := vars["task-id"]

	new_userID, erro := strconv.Atoi(idStr)
	if erro != nil {
		h.log.ERROR("Handler(db-service)", "Create_Task", "Ошибка чтения user_id", nil)
		http.Error(w, "Ошибка чтения id", http.StatusBadRequest)
		return
	}
	new_taskID, errorr := strconv.Atoi(taskStr)
	if errorr != nil {
		h.log.ERROR("Handler(db-service)", "Create_Task", "Ошибка чтения task_id", nil)
		http.Error(w, "Ошибка чтения id", http.StatusBadRequest)
		return
	}

	_, err := h.s.DeleteTask(ctx, new_userID, new_taskID)
	if err != nil {
		if errors.Is(err, apperrors.ErrTaskNotFound) {
			h.log.ERROR("Handler(db-service)", "Delete", "Задача не найдена", &new_userID)
			http.Error(w, "Задача не найдена", http.StatusNotFound)
			return

		}
		h.log.ERROR("Handler(db-service)", "Delete", "Неизвестная ошибка либо упала транзакция", &new_userID)
		http.Error(w, "Ошибка", http.StatusInternalServerError)
		return

	}
	w.WriteHeader(http.StatusOK)

	h.log.INFO("Handler(db-service)", "Delete", "Задача успешно удалена", &new_userID)

}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	ctx := r.Context()
	vars := mux.Vars(r)
	idStr := vars["user_id"]
	taskStr := vars["task-id"]

	new_userID, err := strconv.Atoi(idStr)
	if err != nil {
		h.log.ERROR("Handler(db-service)", "Update", "invalid user_id", nil)
		http.Error(w, "Неверный формат user_id", http.StatusBadRequest)
		return
	}

	new_taskID, err := strconv.Atoi(taskStr)
	if err != nil {
		h.log.ERROR("Handler(db-service)", "Update", "invalid task_id", nil)
		http.Error(w, "Неверный формат task_id", http.StatusBadRequest)
		return
	}

	_, erro := h.s.Update_Task(ctx, new_userID, new_taskID)
	if erro != nil {
		if errors.Is(err, apperrors.ErrTaskNotFound) {
			h.log.ERROR("Handler(db-service)", "Update", "Задача не найдена", &new_userID)
			http.Error(w, "Задача не найдена", http.StatusNotFound)
			return

		}
		h.log.ERROR("Handler(db-service)", "Update", "Неизвестная ошибка либо упала транзакция", &new_userID)
		http.Error(w, "Ошибка", http.StatusInternalServerError)
		return

	}
	w.WriteHeader(http.StatusOK)

	h.log.INFO("Handler(db-service)", "Update", "Задача успешно обновлена", &new_userID)

}

func (h *Handler) GetAllTasks(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	ctx := r.Context()
	vars := mux.Vars(r)
	idStr := vars["user_id"]

	new_ID, err := strconv.Atoi(idStr)
	if err != nil {
		h.log.ERROR("Handler(db-service)", "GetAllTasks", "invalid user_id", nil)
		http.Error(w, "Неверный формат user_id", http.StatusBadRequest)
		return
	}
	task, err := h.s.GetAllTasks(ctx, new_ID)
	if err != nil {
		if errors.Is(err, apperrors.ErrEmptySlice) {
			http.Error(w, "нет задач", http.StatusNotFound)

			return
		}
		http.Error(w, "Ошибка при получении задач", http.StatusInternalServerError)

		return
	}
	w.WriteHeader(http.StatusOK)
	erro := json.NewEncoder(w).Encode(task)
	if erro != nil {
		h.log.ERROR("Handler(db-service)", "Json encoding", "Ошибка при кодировании Task в Json", &new_ID)
		http.Error(w, "Ошибка кодирования json ", http.StatusBadRequest)
		return

	}

}
