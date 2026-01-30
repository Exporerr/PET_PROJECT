package testik

import (
	"context"
	"errors"
	"strings"

	models "github.com/Explorerr/pet_project/pkg/Models"
	apperrors "github.com/Explorerr/pet_project/pkg/app_errors"
)

var ErrTaskNotFound = errors.New("task not found")
var ErrEmptySlice = errors.New("there is no tasks")
var ErrTooFewTasks = errors.New("there are too few tasks")
var ErrInvalidInput = errors.New("invalid input")

type Service_mock struct {
	Log         *models.Request_Login
	Register    *models.Request_Register
	User        *models.User
	Task        *models.Task
	Tasks       *[]models.Task
	Title       string
	Description string
	ResultID    int
	Err         error
}

func (s *Service_mock) DeleteTask(ctx context.Context, userID, taskID int) (bool, error) {
	if s.Err != nil {
		if errors.Is(s.Err, ErrTaskNotFound) {

			return false, ErrTaskNotFound
		}
		return false, s.Err

	}
	return true, nil

}

func (s *Service_mock) Create_Task(ctx context.Context, task *models.Request_Task, user_id int) (*models.Task, error) {

	if strings.TrimSpace(task.Title) == "" || task.Description == "" {
		return nil, apperrors.ErrInvalidInput
	}
	if s.Err != nil {
		if errors.Is(s.Err, apperrors.ErrWentWrong) {
			return nil, apperrors.ErrWentWrong
		}
		return nil, s.Err
	}

	return s.Task, nil

}
func (s *Service_mock) Update_Task(ctx context.Context, user_id int, task_id int) (bool, error) {
	if s.Err != nil {
		if errors.Is(s.Err, ErrTaskNotFound) {

			return false, ErrTaskNotFound
		}
		return false, s.Err

	}
	return true, nil

}
func (s *Service_mock) GetAllTasks(ctx context.Context, user_id int) ([]models.Task, error) {
	if len(*s.Tasks) == 0 {

		return nil, s.Err
	}

	return *s.Tasks, nil
}
func (s *Service_mock) Create_New_user(ctx context.Context, user models.Request_Register) error {

	if s.Err != nil {
		if errors.Is(s.Err, apperrors.ErrUserAlreadyExists) {

			return apperrors.ErrUserAlreadyExists
		}

		return s.Err

	}
	return nil

}
func (s *Service_mock) Login(ctx context.Context, email string) (*models.User, error) {

	if s.Err != nil {

		return nil, s.Err

	}
	return s.User, nil

}

func (s *Service_mock) GetTitle() string {
	return s.Title
}
func (s *Service_mock) GetDesc() string {
	return s.Title
}
