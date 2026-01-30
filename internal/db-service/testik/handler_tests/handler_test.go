package handlers_test

import (
	"bytes"
	"io"

	"encoding/json"
	"errors"

	//"io"
	"net/http"
	"net/http/httptest"

	"testing"

	handler "github.com/Explorerr/pet_project/internal/db-service/Handler"
	"github.com/Explorerr/pet_project/internal/db-service/testik"

	kafkalogger "github.com/Explorerr/pet_project/pkg/Kafka_logger"
	models "github.com/Explorerr/pet_project/pkg/Models"
)

func Test_for_Handler_Create_Task(t *testing.T) {
	tests := []struct {
		name       string
		service    *testik.Service_mock
		wantStatus int
		wantID     int64
		wantErr    string
		user_id    string
	}{
		{
			name:       "Хороший тест(Create-Task)",
			service:    &testik.Service_mock{Title: "Тест", Description: "проверка", ResultID: 100, Err: nil},
			wantStatus: http.StatusCreated,
			wantID:     100,
			user_id:    "435252562",
		},
		{
			name:       "Плохой тест(Create-Task) – пустые поля",
			service:    &testik.Service_mock{Title: "", Description: "", Err: testik.ErrInvalidInput},
			wantStatus: http.StatusBadRequest,
			wantErr:    testik.ErrInvalidInput.Error(),
			user_id:    "435252562",
		},

		{
			name:       "Ошибка сервиса(Create-Task)",
			service:    &testik.Service_mock{Title: "X", Description: "Y", Err: errors.New("db failed")},
			wantStatus: http.StatusInternalServerError,
			wantErr:    "db failed",
			user_id:    "435252562",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// формируем JSON тела запроса
			task := models.Task{Title: tt.service.Title, Description: tt.service.Description}
			jsonBody, _ := json.Marshal(task)
			req := httptest.NewRequest(http.MethodPost, "/?user_id="+tt.user_id, bytes.NewReader(jsonBody))

			req.Header.Set("Content-Type", "application/json")

			// создаём хендлер с мок-сервисом
			w := httptest.NewRecorder()

			handler := handler.NewHandler(&kafkalogger.Logger_For_Tests{}, tt.service)
			handler.Create_Task(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}

			bodyStr := string(bodyBytes)

			// Проверяем статус
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, resp.StatusCode)
				t.Errorf("got message: %s", bodyStr)
			}

			// Если ожидаем ошибку, проверяем тело
			if tt.wantErr != "" && bodyStr != "" && bodyStr != tt.wantErr+"\n" {
				t.Errorf("expected error message %q, got %q", tt.wantErr, bodyStr)
			}

		})

	}
}
