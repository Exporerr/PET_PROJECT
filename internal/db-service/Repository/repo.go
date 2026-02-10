package repository

import (
	"context"

	models "github.com/Explorerr/pet_project/pkg/Models"
)

type Repository interface {
	Create_Task(ctx context.Context, task *models.Request_Task, user_id int) (*models.Task, error)
	Update_Task(ctx context.Context, user_id int, task_id int) error
	Get_Tasks(user_id int, ctx context.Context) ([]models.Task, error)
	Login(ctx context.Context, email string) (*models.User, error)
	Create_New_User(ctx context.Context, user models.Request_Register) error
	DeleteTask(ctx context.Context, userID, taskID int) error
}

func NewRepo(db *Storage) Repository {
	return db
}
