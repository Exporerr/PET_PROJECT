package repository

import (
	"context"
	"errors"
	"time"

	"fmt"

	kafkalogger "github.com/Explorerr/pet_project/pkg/Kafka_logger"
	models "github.com/Explorerr/pet_project/pkg/Models"
	apperrors "github.com/Explorerr/pet_project/pkg/app_errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
			s.log.ERROR(" Create_New_User", "Создание пользователя", fmt.Sprintf("пользователь с email %s уже существует", user.Email), nil)
			return apperrors.ErrUserAlreadyExists
		}
		return err

	}
	s.log.INFO(" Create_New_User", "Создание пользователя", fmt.Sprintf("пользователь с email:|%s|, id:|%d| успешно создан", user.Email, &id), &id)
	return nil

}

func (s *Storage) Create_Task(ctx context.Context, task *models.Request_Task, user_id int) (*models.Task, error) {
	const (
		maxRetries = 3
	)
	for i := 0; i < maxRetries; i++ {
		tx, err := s.db.Begin(ctx)
		if err != nil {
			s.log.ERROR("storage", "Create_Task", fmt.Sprintf("Не удалось начать транзакцию: %v", err), &user_id)
			return nil, apperrors.ErrTransactionNotInit
		}

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
			_ = tx.Rollback(ctx)
			if !isRetryable(erro) || i == maxRetries-1 {
				s.log.ERROR("Create_Task(Storage)", "Создание задачи", fmt.Sprintf("ошибка при создании задачи: %v", erro), &user_id)
				return nil, erro
			}

			retryDelay(i)
			continue

		}
		if err = tx.Commit(ctx); err != nil {
			_ = tx.Rollback(ctx)
			if !isRetryable(err) || i == maxRetries-1 {
				s.log.ERROR("storage", "Create_Task", fmt.Sprintf("Ошибка коммита транзакции: %v", err), &user_id)
				return nil, fmt.Errorf("commit failed: %w", err)
			}

			retryDelay(i)
			continue
		}
		s.log.INFO("Create_Task(Storage)", "Создание задачи", fmt.Sprintf("Задача успешно создана , ID ЗАДАЧИ: %d", Resp.ID), &user_id)

		return &Resp, nil

	}
	return nil, apperrors.ErrWentWrong

}

func (s *Storage) DeleteTask(ctx context.Context, user_id int, task_id int) error {
	const (
		maxRetries = 3
	)
	for i := 0; i < maxRetries; i++ {
		if ctx.Err() != nil {

			return ctx.Err()
		}
		tx, err := s.db.Begin(ctx)
		if err != nil {
			s.log.ERROR("storage", "Delete_Task", fmt.Sprintf("Не удалось начать транзакцию: %v", err), &user_id)
			return apperrors.ErrTransactionNotInit

		}

		query := `DELETE FROM tasks WHERE user_id=$1 AND id=$2`
		tag, err := tx.Exec(ctx, query, user_id, task_id)
		if err != nil {
			_ = tx.Rollback(ctx)
			if !isRetryable(err) || i == maxRetries-1 {
				s.log.ERROR("storage", "Delete_Task", fmt.Sprintf("Ошибка выполнения SQL запроса: %v", err), &user_id)
				return apperrors.ErrWentWrong
			}

			retryDelay(i)
			continue
		}

		if tag.RowsAffected() == 0 {
			tx.Rollback(ctx)
			s.log.INFO("storage", "Delete_Task", "Задача не найдена", &user_id)
			return apperrors.ErrTaskNotFound
		}

		if err = tx.Commit(ctx); err != nil {
			_ = tx.Rollback(ctx)
			if !isRetryable(err) || i == maxRetries-1 {
				s.log.ERROR("storage", "Delete_Task", fmt.Sprintf("Ошибка коммита транзакции: %v", err), &user_id)
				return fmt.Errorf("commit failed: %w", err)
			}

			retryDelay(i)
			continue

		}
		s.log.INFO("storage", "Delete_Task", "Задача успешно удалена", &user_id)
		return nil
	}
	return apperrors.ErrWentWrong
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

func (s *Storage) Update_Task(ctx context.Context, user_id int, task_id int) error {
	const (
		maxRetries = 3
	)
	for i := 0; i < maxRetries; i++ {
		if ctx.Err() != nil {

			return ctx.Err()
		}

		tx, err := s.db.Begin(ctx)
		if err != nil {
			s.log.ERROR("storage", "Update_task", fmt.Sprintf("Не удалось начать транзакцию: %v", err), &user_id)
			return apperrors.ErrTransactionNotInit

		}

		query := `UPDATE tasks SET status = 'done' WHERE user_id = $1 AND id = $2`
		tag, err := tx.Exec(ctx, query, user_id, task_id)
		if err != nil {
			_ = tx.Rollback(ctx)
			if !isRetryable(err) || i == maxRetries-1 {
				s.log.ERROR("storage", "Update_task", fmt.Sprintf("Ошибка выполнения SQL запроса: %v", err), &user_id)
				return apperrors.ErrWentWrong
			}
			retryDelay(i)
			continue
		}

		if tag.RowsAffected() == 0 {
			tx.Rollback(ctx)
			s.log.INFO("storage", "Update_task", "Задача не найдена", &user_id)
			return apperrors.ErrTaskNotFound
		}

		if err = tx.Commit(ctx); err != nil {
			_ = tx.Rollback(ctx)
			if !isRetryable(err) || i == maxRetries-1 {
				s.log.ERROR("storage", "Update_task", fmt.Sprintf("Ошибка коммита транзакции: %v", err), &user_id)
				return fmt.Errorf("commit failed: %w", err)
			}

			retryDelay(i)
			continue

		}
		s.log.INFO("storage", "Update_task", "Задача успешно выполнена", &user_id)
		return nil
	}
	return apperrors.ErrWentWrong
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

// вспомогательные  функции

func isRetryable(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "40P01", // deadlock_detected
			"55P03", // lock_not_available
			"40001": // serialization failure
			return true
		}
	}
	return false
}

func retryDelay(i int) {
	base := 100 * time.Millisecond
	time.Sleep(base * time.Duration(1<<i))
}
