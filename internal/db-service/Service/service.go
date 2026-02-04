package service

import (
	"context"
	"errors"
	"fmt"

	repository "github.com/Explorerr/pet_project/internal/db-service/Repository"
	kafkalogger "github.com/Explorerr/pet_project/pkg/Kafka_logger"
	models "github.com/Explorerr/pet_project/pkg/Models"
	apperrors "github.com/Explorerr/pet_project/pkg/app_errors"
)

type Service struct {
	repo repository.Repository
	log  kafkalogger.LoggerInterface
}

func New_Service(repo repository.Repository, log kafkalogger.LoggerInterface) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) Create_New_user(ctx context.Context, user models.Request_Register) error {

	err := s.repo.Create_New_User(ctx, user)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserAlreadyExists) {
			s.log.ERROR("Service(db-service)", "Create_New_User", "Пользователь уже существует", nil)
			return apperrors.ErrUserAlreadyExists
		}
		s.log.ERROR("Service(db-service)", "Crearte_New_User", fmt.Sprintf("Что-то пошло не так : %v", err), nil)
		return err

	}
	s.log.ERROR("Service(db-service)", "Crearte_New_User", "Сервис успешно отработал", nil)
	return nil

}

func (s *Service) Login(ctx context.Context, email string) (*models.User, error) {
	User, err := s.repo.Login(ctx, email)
	if err != nil {

		if errors.Is(err, apperrors.ErrUserNotExist) {
			s.log.ERROR("Service(db-service)", "Login", fmt.Sprintf("Такого пользователя не сущетствует: %v", err), &User.ID)
			return nil, apperrors.ErrUserNotExist
		}
		s.log.ERROR("Service(db-service)", "Login", fmt.Sprintf("Ошибка из Репозитория : %v ", err), &User.ID)
		return nil, err

	}
	s.log.ERROR("Service(db-service)", "Login", "Сервис успешно отработал", nil)
	return User, nil

}

func (s *Service) GetAllTasks(ctx context.Context, user_id int) ([]models.Task, error) {
	tasks, err := s.repo.Get_Tasks(user_id, ctx)
	if err != nil {
		s.log.ERROR("Service(db-service)", "Get-Tasks(service)", fmt.Sprintf("repo.GetAllTasks failed: %v", err), &user_id)
		return nil, err
	}

	if len(tasks) == 0 {
		s.log.INFO("Service(db-service)", "Get-Tasks(service)", fmt.Sprintf("Empty slice error: %v", err), &user_id)

		return nil, apperrors.ErrEmptySlice
	}
	s.log.INFO("Service(db-service)", "Get-Tasks(service)", "GetAllTask(db-service) executed successfully!", &user_id)

	s.log.DEBUG("Service(db-service)", "Get-Tasks(service)", "Success", &user_id)

	return tasks, nil
}

func (s *Service) Create_Task(ctx context.Context, task *models.Request_Task, user_id int) (*models.Task, error) {

	taskModel, err := s.repo.Create_Task(ctx, task, user_id)
	if err == nil {
		s.log.ERROR("Service(db-service)", "Create_Task", fmt.Sprintf("Не удалось создать задачу по ошибке: %v", err), &user_id)
		return nil, err

	}
	s.log.INFO("Service(db-service)", "Create_Task", "Задача успешно создана", &user_id)
	return taskModel, nil

}

func (s *Service) DeleteTask(ctx context.Context, userID, taskID int) (bool, error) {

	deleted, err := s.repo.DeleteTask(ctx, userID, taskID)
	if err != nil {
		if errors.Is(err, apperrors.ErrTaskNotFound) {
			s.log.ERROR("Service(db-service)", "Delete_Task", "Задачи не существует", &userID)
			return false, err
		}

		s.log.INFO("Service(db-service)", "Delete_Task", "Ошибка удаления задачи", &userID)
		return false, err

	}
	s.log.INFO("Service(db-service)", "Delete_Task", "Задача успешно удалена", &userID)
	return deleted, nil

}

func (s *Service) Update_Task(ctx context.Context, user_id int, task_id int) (bool, error) {

	updated, err := s.repo.Update_Task(ctx, user_id, task_id)
	if err != nil {
		if errors.Is(err, apperrors.ErrTaskNotFound) {
			s.log.ERROR("Service(db-service)", "Update_Task", "Задачи не существует", &user_id)
			return false, err
		}
		s.log.ERROR("Service(db-service)", "Update_Task", "Ошибка удаления задачи", &user_id)
		return false, err
	}

	s.log.INFO("Service(db-service)", "Update_Task", "Задача успешно обновлена", &user_id)
	return updated, nil
}
