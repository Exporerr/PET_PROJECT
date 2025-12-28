package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	kafkalogger "github.com/Explorerr/pet_project/pkg/Kafka_logger"
	models "github.com/Explorerr/pet_project/pkg/Models"
	apperrors "github.com/Explorerr/pet_project/pkg/app_errors"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	log        kafkalogger.Logger
}

func NewClient(baseURL string, log kafkalogger.Logger) *Client {
	return &Client{
		httpClient: &http.Client{},
		baseURL:    baseURL,
		log:        log,
	}
}

func (cli *Client) Register(user models.Request_Register, ctx context.Context) error {
	url := fmt.Sprintf("%s/tasks/register", cli.baseURL)
	cli.log.DEBUG("Client(api-service)", "Register", fmt.Sprintf("Отправка запроса регистрации на: %s", url), nil)
	body, _ := json.Marshal(user)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := cli.httpClient.Do(req)
	if err != nil {
		cli.log.ERROR("Client(api-service)", "Register", fmt.Sprintf("Ошибка HTTP запроса: %v", err), nil)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		cli.log.ERROR("Client(api-service)", "Register", fmt.Sprintf("HTTP статус код: %d - Пользователь уже существует", resp.StatusCode), nil)
		return apperrors.ErrUserAlreadyExists
	} else if resp.StatusCode != http.StatusCreated {
		cli.log.ERROR("Client(api-service)", "Register", fmt.Sprintf("Неожиданный HTTP статус код: %d", resp.StatusCode), nil)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	cli.log.INFO("Client(api-service)", "Register", fmt.Sprintf("Регистрация успешна, HTTP статус код: %d", resp.StatusCode), nil)
	return nil

}

func (cli *Client) Login(login models.Request_Login, ctx context.Context) (*models.User, error) {
	url := fmt.Sprintf("%s/tasks/login", cli.baseURL)
	cli.log.DEBUG("Client(api-service)", "Login", fmt.Sprintf("Отправка запроса логина на: %s", url), nil)
	body, _ := json.Marshal(login)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := cli.httpClient.Do(req)
	if err != nil {
		cli.log.ERROR("Client(api-service)", "Login", fmt.Sprintf("Ошибка HTTP запроса: %v", err), nil)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		cli.log.ERROR("Client(api-service)", "Login", fmt.Sprintf("HTTP статус код: %d - Пользователь не существует", resp.StatusCode), nil)
		return nil, apperrors.ErrUserNotExist
	} else if resp.StatusCode == http.StatusInternalServerError {
		cli.log.ERROR("Client(api-service)", "Login", fmt.Sprintf("HTTP статус код: %d - Внутренняя ошибка сервера", resp.StatusCode), nil)
		return nil, apperrors.ErrWentWrong
	} else if resp.StatusCode != http.StatusOK {
		cli.log.ERROR("Client(api-service)", "Login", fmt.Sprintf("Неожиданный HTTP статус код: %d", resp.StatusCode), nil)
		return nil, errors.New("internal errors")
	}
	var user models.User
	erro := json.NewDecoder(resp.Body).Decode(&user)
	if erro != nil {
		cli.log.ERROR("Client(api-service)", "Login", fmt.Sprintf("Ошибка декодирования ответа: %v, HTTP статус код: %d", erro, resp.StatusCode), nil)
		return nil, erro

	}
	cli.log.INFO("Client(api-service)", "Login", fmt.Sprintf("Логин успешен, HTTP статус код: %d", resp.StatusCode), &user.ID)
	return &user, nil

}

func (cli *Client) Create_Task(id int, task models.Request_Task, ctx context.Context) (*models.Task, error) {
	url := fmt.Sprintf("%s/tasks?user_id=%d", cli.baseURL, id)
	cli.log.DEBUG("Client(api-service)", "Create_Task", fmt.Sprintf("Отправка запроса создания задачи на: %s для пользователя: %d", url, id), &id)
	body, _ := json.Marshal(task)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := cli.httpClient.Do(req)
	if err != nil {
		cli.log.ERROR("Client(api-service)", "Create_Task", fmt.Sprintf("Ошибка HTTP запроса: %v", err), &id)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		cli.log.ERROR("Client(api-service)", "Create_Task", fmt.Sprintf("Неожиданный HTTP статус код: %d", resp.StatusCode), &id)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	var task_resp models.Task
	erro := json.NewDecoder(resp.Body).Decode(&task_resp)
	if erro != nil {
		cli.log.ERROR("Client(api-service)", "Create_Task", fmt.Sprintf("Ошибка декодирования ответа: %v, HTTP статус код: %d", erro, resp.StatusCode), &id)
		return nil, erro
	}
	cli.log.INFO("Client(api-service)", "Create_Task", fmt.Sprintf("Задача успешно создана, HTTP статус код: %d, ID задачи: %d", resp.StatusCode, task_resp.ID), &id)
	return &task_resp, nil

}

func (cli *Client) GetAllTasks(id int, ctx context.Context) ([]models.Task, error) {
	url := fmt.Sprintf("%s/tasks/%d", cli.baseURL, id)
	cli.log.DEBUG("Client(api-service)", "GetAllTasks", fmt.Sprintf("Отправка запроса получения задач на: %s для пользователя: %d", url, id), &id)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := cli.httpClient.Do(req)
	if err != nil {
		cli.log.ERROR("Client(api-service)", "GetAllTasks", fmt.Sprintf("Ошибка HTTP запроса: %v", err), &id)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		cli.log.ERROR("Client(api-service)", "GetAllTasks", fmt.Sprintf("HTTP статус код: %d - У пользователя нет задач", resp.StatusCode), &id)
		return nil, apperrors.ErrEmptySlice
	} else if resp.StatusCode == http.StatusInternalServerError {
		cli.log.ERROR("Client(api-service)", "GetAllTasks", fmt.Sprintf("HTTP статус код: %d - Внутренняя ошибка сервера", resp.StatusCode), &id)
		return nil, errors.New("internal errors")
	} else if resp.StatusCode != http.StatusOK {
		cli.log.ERROR("Client(api-service)", "GetAllTasks", fmt.Sprintf("Неожиданный HTTP статус код: %d", resp.StatusCode), &id)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	var tasks []models.Task
	erro := json.NewDecoder(resp.Body).Decode(&tasks)
	if erro != nil {
		cli.log.ERROR("Client(api-service)", "GetAllTasks", fmt.Sprintf("Ошибка декодирования ответа: %v, HTTP статус код: %d", erro, resp.StatusCode), &id)
		return nil, erro
	}
	cli.log.INFO("Client(api-service)", "GetAllTasks", fmt.Sprintf("Успешно получено задач: %d, HTTP статус код: %d", len(tasks), resp.StatusCode), &id)
	return tasks, nil

}

func (cli *Client) Delete(ctx context.Context, user_id int, task_id int) error {
	url := fmt.Sprintf("%s/tasks/del/%d/%d", cli.baseURL, user_id, task_id)
	cli.log.DEBUG("Client(api-service)", "Delete", fmt.Sprintf("Отправка запроса удаления задачи на: %s", url), &user_id)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		cli.log.ERROR("Client(api-service)", "Delete", fmt.Sprintf("Ошибка создания запроса: %v", err), &user_id)
		return err
	}

	resp, err := cli.httpClient.Do(req)
	if err != nil {
		cli.log.ERROR("Client(api-service)", "Delete", fmt.Sprintf("Ошибка HTTP запроса: %v", err), &user_id)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		cli.log.ERROR("Client(api-service)", "Delete", fmt.Sprintf("HTTP статус код: %d - Задача не найдена", resp.StatusCode), &user_id)
		return apperrors.ErrTaskNotFound
	} else if resp.StatusCode == http.StatusInternalServerError {
		cli.log.ERROR("Client(api-service)", "Delete", fmt.Sprintf("HTTP статус код: %d - Внутренняя ошибка сервера", resp.StatusCode), &user_id)
		return apperrors.ErrWentWrong
	} else if resp.StatusCode != http.StatusOK {
		cli.log.ERROR("Client(api-service)", "Delete", fmt.Sprintf("Неожиданный HTTP статус код: %d", resp.StatusCode), &user_id)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	cli.log.INFO("Client(api-service)", "Delete", fmt.Sprintf("Задача успешно удалена, HTTP статус код: %d, task_id: %d", resp.StatusCode, task_id), &user_id)
	return nil
}

func (cli *Client) Update(ctx context.Context, user_id int, task_id int) error {
	url := fmt.Sprintf("%s/tasks/up/%d/%d", cli.baseURL, user_id, task_id)
	cli.log.DEBUG("Client(api-service)", "Update", fmt.Sprintf("Отправка запроса обновления задачи на: %s", url), &user_id)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, nil)
	if err != nil {
		cli.log.ERROR("Client(api-service)", "Update", fmt.Sprintf("Ошибка создания запроса: %v", err), &user_id)
		return err
	}

	resp, err := cli.httpClient.Do(req)
	if err != nil {
		cli.log.ERROR("Client(api-service)", "Update", fmt.Sprintf("Ошибка HTTP запроса: %v", err), &user_id)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		cli.log.ERROR("Client(api-service)", "Update", fmt.Sprintf("HTTP статус код: %d - Задача не найдена", resp.StatusCode), &user_id)
		return apperrors.ErrTaskNotFound
	} else if resp.StatusCode == http.StatusInternalServerError {
		cli.log.ERROR("Client(api-service)", "Update", fmt.Sprintf("HTTP статус код: %d - Внутренняя ошибка сервера", resp.StatusCode), &user_id)
		return apperrors.ErrWentWrong
	} else if resp.StatusCode != http.StatusOK {
		cli.log.ERROR("Client(api-service)", "Update", fmt.Sprintf("Неожиданный HTTP статус код: %d", resp.StatusCode), &user_id)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	cli.log.INFO("Client(api-service)", "Update", fmt.Sprintf("Задача успешно обновлена, HTTP статус код: %d, task_id: %d", resp.StatusCode, task_id), &user_id)
	return nil
}
