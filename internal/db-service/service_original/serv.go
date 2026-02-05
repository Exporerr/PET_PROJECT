package serviceoriginal

import (
	"context"

	service "github.com/Explorerr/pet_project/internal/db-service/Service"
	models "github.com/Explorerr/pet_project/pkg/Models"
)

type User_Interface interface {
	Update_Task(ctx context.Context, user_id int, task_id int) error
	Create_Task(ctx context.Context, task *models.Request_Task, user_id int) (*models.Task, error)
	GetAllTasks(ctx context.Context, user_id int) ([]models.Task, error)
	Login(ctx context.Context, email string) (*models.User, error)
	Create_New_user(ctx context.Context, user models.Request_Register) error
	DeleteTask(ctx context.Context, userID, taskID int) error
}

func New_User_service(s *service.Service) User_Interface {
	return s
}
