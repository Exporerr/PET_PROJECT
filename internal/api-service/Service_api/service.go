package serviceapi

import (
	"context"
	"errors"
	"fmt"

	"os"
	"strconv"
	"time"

	"strings"

	client "github.com/Explorerr/pet_project/internal/api-service/Client"
	kafkalogger "github.com/Explorerr/pet_project/pkg/Kafka_logger"
	models "github.com/Explorerr/pet_project/pkg/Models"
	apperrors "github.com/Explorerr/pet_project/pkg/app_errors"
	"github.com/golang-jwt/jwt/v5"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	cli client.Client
	log kafkalogger.Logger
}

func NewService_api(cli client.Client, log kafkalogger.Logger) *Service {
	return &Service{cli: cli, log: log}
}

func (s *Service) Register(user models.Request_Register, ctx context.Context) error {
	s.log.DEBUG("Service(api-service)", "Register", fmt.Sprintf("Начало регистрации пользователя: %s", user.Email), nil)
	if strings.TrimSpace(user.Username) == "" || strings.TrimSpace(user.Email) == "" || strings.TrimSpace(user.Password) == "" {
		s.log.ERROR("Service(api-service)", "Register", "Пустые поля в запросе регистрации", nil)
		return apperrors.ErrInvalidInput
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		s.log.ERROR("Service(api-service)", "Register", fmt.Sprintf("Ошибка хеширования пароля: %v", err), nil)
		return fmt.Errorf("failed to hash password: %w", err)
	}
	user.Password = string(hash)

	if err := s.cli.Register(user, ctx); err != nil {
		if errors.Is(err, apperrors.ErrUserAlreadyExists) {
			s.log.ERROR("Service(api-service)", "Register", fmt.Sprintf("Пользователь уже существует: %s", user.Email), nil)
			return apperrors.ErrUserAlreadyExists
		}
		s.log.ERROR("Service(api-service)", "Register", fmt.Sprintf("Ошибка при регистрации через клиент: %v", err), nil)
		return err
	}
	s.log.INFO("Service(api-service)", "Register", fmt.Sprintf("Пользователь успешно зарегистрирован: %s", user.Email), nil)
	return nil

}

func (s *Service) LoginService(login models.Request_Login, ctx context.Context) (string, *models.User, error) {
	s.log.DEBUG("Service(api-service)", "LoginService", fmt.Sprintf("Начало авторизации пользователя: %s", login.Email), nil)
	user, err := s.cli.Login(login, ctx)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserNotExist) {
			s.log.ERROR("Service(api-service)", "LoginService", fmt.Sprintf("Такого пользователя не существует: %v", err), nil)
			return "", nil, apperrors.ErrUserNotExist
		}
		s.log.ERROR("Service(api-service)", "LoginService", fmt.Sprintf("Ошибка при логине через клиент: %v", err), nil)
		return "", nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password_Hash), []byte(login.Password)); err != nil {
		s.log.ERROR("Service(api-service)", "LoginService", fmt.Sprintf("Неверный пароль для пользователя: %s", login.Email), &user.ID)
		return "", nil, fmt.Errorf("invalid password")
	}

	JWTSecret := os.Getenv("JWT_SECRET")
	if JWTSecret == "" {
		s.log.ERROR("Service(api-service)", "LoginService", "JWT_SECRET не установлен в переменных окружения", &user.ID)
		return "", nil, fmt.Errorf("JWT_SECRET not set")
	}

	expireHoursStr := os.Getenv("JWT_EXPIRE_HOURS")
	expireHours, err := strconv.Atoi(expireHoursStr)
	if err != nil {
		s.log.ERROR("Service(api-service)", "LoginService", fmt.Sprintf("Ошибка парсинга JWT_EXPIRE_HOURS: %v", err), &user.ID)
		return "", nil, err
	}
	claims := models.MyClaims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(JWTSecret))
	if err != nil {
		s.log.ERROR("Service(api-service)", "LoginService", fmt.Sprintf("Ошибка создания JWT токена: %v", err), &user.ID)
		return "", nil, err
	}

	s.log.INFO("Service(api-service)", "LoginService", fmt.Sprintf("Пользователь успешно авторизован: %s", login.Email), &user.ID)
	return signedToken, user, nil
}

func (s *Service) Create_Task(id int, task models.Request_Task, ctx context.Context) (*models.Task, error) {
	s.log.DEBUG("Service(api-service)", "Create_Task", fmt.Sprintf("Начало создания задачи для пользователя: %d", id), &id)
	if id == 0 {
		s.log.ERROR("Service(api-service)", "Create_Task", "ID пользователя пустой", nil)
		return nil, fmt.Errorf("%w: id пустой", apperrors.ErrInvalidInput)
	}
	if strings.TrimSpace(task.Title) == "" {
		s.log.ERROR("Service(api-service)", "Create_Task", "Заголовок задачи пустой", &id)
		return nil, fmt.Errorf("%w: title cannot be empty", apperrors.ErrInvalidInput)
	}
	if strings.TrimSpace(task.Description) == "" {
		s.log.ERROR("Service(api-service)", "Create_Task", "Описание задачи пустое", &id)
		return nil, fmt.Errorf("%w: description cannot be empty", apperrors.ErrInvalidInput)
	}
	resptask, err := s.cli.Create_Task(id, task, ctx)
	if err != nil {
		s.log.ERROR("Service(api-service)", "Create_Task", fmt.Sprintf("Ошибка при создании задачи через клиент: %v", err), &id)
		return nil, err
	}
	s.log.INFO("Service(api-service)", "Create_Task", fmt.Sprintf("Задача успешно создана, ID задачи: %d", resptask.ID), &id)
	return resptask, nil
}

func (s *Service) GetAllTasks(id int, ctx context.Context) ([]models.Task, error) {
	s.log.DEBUG("Service(api-service)", "GetAllTasks", fmt.Sprintf("Начало получения задач для пользователя: %d", id), &id)
	if id == 0 {
		s.log.ERROR("Service(api-service)", "GetAllTasks", "ID пользователя пустой", nil)
		return nil, fmt.Errorf("%w: id пустой", apperrors.ErrInvalidInput)
	}
	tasks, err := s.cli.GetAllTasks(id, ctx)
	if err != nil {
		if errors.Is(err, apperrors.ErrEmptySlice) {
			s.log.INFO("Service(api-service)", "GetAllTasks", fmt.Sprintf("У пользователя нет задач: %d", id), &id)
			return nil, apperrors.ErrEmptySlice
		}
		s.log.ERROR("Service(api-service)", "GetAllTasks", fmt.Sprintf("Ошибка при получении задач через клиент: %v", err), &id)
		return nil, err
	}

	s.log.INFO("Service(api-service)", "GetAllTasks", fmt.Sprintf("Успешно получено задач: %d для пользователя: %d", len(tasks), id), &id)

	return tasks, nil
}

func (s *Service) Update(ctx context.Context, user_id int, task_id int) error {
	s.log.DEBUG("Service(api-service)", "Update", fmt.Sprintf("Начало обновления задачи: %d для пользователя: %d", task_id, user_id), &user_id)

	if user_id == 0 || task_id == 0 {
		s.log.ERROR("Service(api-service)", "Update", "Пустые ID пользователя или задачи", nil)
		return apperrors.ErrInvalidInput
	}
	err := s.cli.Update(ctx, user_id, task_id)
	if err != nil {
		if errors.Is(err, apperrors.ErrTaskNotFound) {
			s.log.ERROR("Service(api-service)", "Update", fmt.Sprintf("Задача не найдена: task_id=%d, user_id=%d", task_id, user_id), &user_id)
			return apperrors.ErrTaskNotFound
		} else if errors.Is(err, apperrors.ErrWentWrong) {
			s.log.ERROR("Service(api-service)", "Update", fmt.Sprintf("Ошибка при обновлении задачи через клиент: %v", err), &user_id)
			return apperrors.ErrWentWrong
		}
		s.log.ERROR("Service(api-service)", "Update", fmt.Sprintf("Неизвестная ошибка при обновлении: %v", err), &user_id)
		return err
	}
	s.log.INFO("Service(api-service)", "Update", fmt.Sprintf("Задача успешно обновлена: task_id=%d", task_id), &user_id)
	return nil
}

func (s *Service) Delete(ctx context.Context, user_id int, task_id int) error {
	s.log.DEBUG("Service(api-service)", "Delete", fmt.Sprintf("Начало удаления задачи: %d для пользователя: %d", task_id, user_id), &user_id)
	if user_id == 0 || task_id == 0 {
		s.log.ERROR("Service(api-service)", "Delete", "Пустые ID пользователя или задачи", nil)
		return apperrors.ErrInvalidInput
	}
	err := s.cli.Delete(ctx, user_id, task_id)
	if err != nil {
		if errors.Is(err, apperrors.ErrTaskNotFound) {
			s.log.ERROR("Service(api-service)", "Delete", fmt.Sprintf("Задача не найдена: task_id=%d, user_id=%d", task_id, user_id), &user_id)
			return apperrors.ErrTaskNotFound
		} else if errors.Is(err, apperrors.ErrWentWrong) {
			s.log.ERROR("Service(api-service)", "Delete", fmt.Sprintf("Ошибка при удалении задачи через клиент: %v", err), &user_id)
			return apperrors.ErrWentWrong
		}
		s.log.ERROR("Service(api-service)", "Delete", fmt.Sprintf("Неизвестная ошибка при удалении: %v", err), &user_id)
		return err
	}
	s.log.INFO("Service(api-service)", "Delete", fmt.Sprintf("Задача успешно удалена: task_id=%d", task_id), &user_id)
	return nil
}
