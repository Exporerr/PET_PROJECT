package repository

import (
	"context"
	"errors"

	"fmt"

	kafkalogger "github.com/Explorerr/pet_project/pkg/Kafka_logger"
	models "github.com/Explorerr/pet_project/pkg/Models"
	apperrors "github.com/Explorerr/pet_project/pkg/app_errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	db  *pgxpool.Pool
	log kafkalogger.LoggerInterface
}

// НОВЫЙ ПУЛ СОЕДИНЕНИЙ
func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

// НОВОЕ ХРАНИЛИЩЕ
func NewUserPool(pool *pgxpool.Pool, kafka_logger kafkalogger.LoggerInterface) *Storage {
	return &Storage{db: pool, log: kafka_logger}
}

func (s *Storage) Create_Index(ctx context.Context) error { // ЗАПУСТИТЬ ПОСЛЕ СОЗДАНИЯ STORAGE
	indexes := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks(user_id)`,
	}
	for _, index := range indexes {
		if _, err := s.db.Exec(ctx, index); err != nil {
			return fmt.Errorf("ошибка при создании индекса: %w", err)
		}
	}

	return nil
}
func (s *Storage) If_notExist(ctx context.Context) {
	query := `CREATE TABLE IF NOT EXISTS tasks (
    id SERIAL PRIMARY KEY,      
    title VARCHAR(255) NOT NULL, 
    description TEXT,             
    status BOOLEAN DEFAULT FALSE, 
    created_at TIMESTAMP DEFAULT NOW()  
);`
	s.db.Exec(ctx, query)

}

// РЕГИСТРАЦИЯ ПОЛЬЗОВАТЕЛЯ
func (s *Storage) Create_New_User(ctx context.Context, user models.Request_Register) error {
	var id int

	query := `
        INSERT INTO users (username ,email, password_hash)
        VALUES ($1, $2, $3)
        ON CONFLICT (email) DO NOTHING
        RETURNING id
    `
	err := s.db.QueryRow(ctx, query,
		user.Username,
		user.Email,
		user.Password,
	).Scan(&id)

	if err != nil {
		if err == pgx.ErrNoRows {
			s.log.ERROR(" Create_New_User", "Создание пользователя", fmt.Sprintf("пользователь с email %s уже существует", user.Email), &id)
			return apperrors.ErrUserAlreadyExists
			//return apperrors.ErrUserAlreadyExists

		}

		return err

	}

	return nil

}

func (s *Storage) Create_Task(ctx context.Context, task *models.Request_Task, user_id int) (*models.Task, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.log.ERROR("storage", "Create_Task", fmt.Sprintf("Не удалось начать транзакцию: %v", err), &user_id)
		return nil, apperrors.ErrTransactionNotInit

	}
	defer tx.Rollback(ctx)
	var Resp models.Task
	query := `
	INSERT INTO tasks ( user_id, title , description)
	VALUES ($1, $2, $3)
	 RETURNING id, title, description, status, created_at,user_id
	
	`
	erro := tx.QueryRow(ctx, query,
		user_id,
		task.Title,
		task.Description).Scan(&Resp.ID, &Resp.Title, &Resp.Description, &Resp.Status, &Resp.CreatedAt, &Resp.UserID)
	if erro != nil {
		s.log.ERROR("Create_Task(Storage)", "Создание задачи", fmt.Sprintf("ошибка при создании задачи: %v", erro), &user_id)

		return nil, erro
	}
	if err = tx.Commit(ctx); err != nil {
		s.log.ERROR("storage", "Create_Task", fmt.Sprintf("Ошибка коммита транзакции: %v", err), &user_id)
		return nil, fmt.Errorf("commit failed: %w", err)
	}
	s.log.INFO("Create_Task(Storage)", "Создание задачи", fmt.Sprintf("Задача успешно создана , ID ЗАДАЧИ: %d", Resp.ID), &user_id)

	return &Resp, nil

}

func (s *Storage) DeleteTask(ctx context.Context, userID, taskID int) (bool, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.log.ERROR("storage", "DeleteTask", fmt.Sprintf("Не удалось начать транзакцию: %v", err), &userID)
		return false, err
	}
	defer tx.Rollback(ctx) // безопасно, если транзакция уже зафиксирована

	query := `DELETE FROM tasks WHERE user_id=$1 AND id=$2`
	tag, err := tx.Exec(ctx, query, userID, taskID)
	if tag.RowsAffected() == 0 {
		s.log.INFO("storage", "DeleteTask", "Задача не найдена", &userID)
		return false, apperrors.ErrTaskNotFound

	}

	if err != nil {

		s.log.ERROR("Storage(db-servic)", "DeleteTask", "Произошла ошибка при удалении задачи", &userID)
		return false, apperrors.ErrWentWrong
	}
	if err = tx.Commit(ctx); err != nil {
		s.log.ERROR("storage", "DeleteTask", fmt.Sprintf("Ошибка коммита транзакции: %v", err), &userID)
		return false, fmt.Errorf("commit failed: %w", err)
	}

	s.log.INFO("storage", "DeleteTask", "Задача успешно удалена", &userID)
	return true, nil
}

func (s *Storage) Get_Tasks(user_id int, ctx context.Context) ([]models.Task, error) {
	var tasks []models.Task

	query := `SELECT id ,title,description, status,created_at FROM tasks WHERE user_id=$1 ORDER BY id ASC LIMIT 10`
	rows, err := s.db.Query(ctx, query, user_id)

	if err != nil {
		s.log.ERROR("Storage", "Get_Tasks", fmt.Sprintf("Ошибка запроса: %v", err), &user_id)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t models.Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.CreatedAt); err != nil {
			s.log.ERROR("Storage", "Получение задачи", fmt.Sprintf("Ошибка скана переменных: %v", err), &user_id)

			return nil, err
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		s.log.ERROR("Storage", "Get_Tasks", fmt.Sprintf("Ошибка при чтении rows: %v", err), &user_id)
		return nil, err
	}

	s.log.INFO("Storage(db-service)", "Получение задач", fmt.Sprintf("Get_Task успешно отработал с пользоватлем :%d", user_id), &user_id)
	return tasks, nil
}

func (s *Storage) Update_Task(ctx context.Context, user_id int, task_id int) (bool, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.log.ERROR("storage", "Update_task", fmt.Sprintf("Не удалось начать транзакцию: %v", err), &user_id)
		return false, apperrors.ErrTransactionNotInit

	}
	defer tx.Rollback(ctx)
	query := `UPDATE tasks SET status = 'done' WHERE user_id = $1 AND id = $2`

	tag, err := tx.Exec(ctx, query, user_id, task_id)

	if err != nil {
		s.log.ERROR("storage", "Update_task", fmt.Sprintf("Ошибка выполнения запроса: %v", err), &user_id)
		return false, apperrors.ErrWentWrong
	}

	if tag.RowsAffected() == 0 {
		s.log.INFO("storage", "update_task", "Задача не найдена", &user_id)
		return false, apperrors.ErrTaskNotFound
	}

	if err = tx.Commit(ctx); err != nil {
		s.log.ERROR("storage", "Update_task", fmt.Sprintf("Ошибка коммита транзакции: %v", err), &user_id)
		return false, fmt.Errorf("commit failed: %w", err)
	}

	s.log.INFO("storage", "Update_task", "Задача успешно выполнена", &user_id)
	return true, nil

}
func (s *Storage) Login(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT id, username, email, password_hash FROM users WHERE email = $1 LIMIT 1`
	user := &models.User{}
	err := s.db.QueryRow(ctx, query, email).Scan(&user.ID, &user.Username, &user.Email, &user.Password_Hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.log.INFO("storage", "Login", "Пользователь не найден", nil)
			return nil, apperrors.ErrUserNotExist
		}
		s.log.ERROR("storage", "Login", fmt.Sprintf("Ошибка сканирования: %v", err), nil)
		return nil, err
	}
	s.log.INFO("storage", "Login", "Пользователь успешно найден", &user.ID)
	return user, nil
}
