package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"

	//"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"testing"

	handler "github.com/Explorerr/pet_project/internal/db-service/Handler"
	repository "github.com/Explorerr/pet_project/internal/db-service/Repository"
	service "github.com/Explorerr/pet_project/internal/db-service/Service"

	"github.com/Explorerr/pet_project/internal/db-service/testik"
	kafkalogger "github.com/Explorerr/pet_project/pkg/Kafka_logger"
	models "github.com/Explorerr/pet_project/pkg/Models"
	"github.com/stretchr/testify/assert"
)

func Test_Create_Task(t *testing.T) {
	tests := []struct {
		name             string
		mockService      *testik.Service_mock
		userID           string
		body             string
		expectedStatus   int
		expectError      bool
		contentType      string
		expectedBodyPart string
	}{
		{
			name: "успешное создание задачи",
			mockService: &testik.Service_mock{
				Err: nil,
				Task: &models.Task{
					Title:       "DFGDGDG",
					Description: "DDGDGDFG",
				},
			},
			userID:         "5",
			body:           `{"title":"","description":""}`,
			expectedStatus: http.StatusCreated,
			expectError:    false,
			contentType:    "application/json",
		},

		{
			name: "Неверный Content-Type",
			mockService: &testik.Service_mock{
				Err: nil,
			},
			userID:           "5",
			body:             `{"title":"Test","description":"Test"}`,
			expectedStatus:   http.StatusUnsupportedMediaType,
			expectError:      true,
			contentType:      "text/plain",
			expectedBodyPart: "Content-Type должен быть application/json",
		},
		{
			name: "Ошибка декодирования JSON",
			mockService: &testik.Service_mock{
				Err: nil,
			},
			userID:           "5",
			body:             ` {invalid json`,
			expectedStatus:   http.StatusBadRequest,
			expectError:      true,
			contentType:      "application/json",
			expectedBodyPart: "Ошибка декодирования JSON тела",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			req := httptest.NewRequest(http.MethodPost, "/?user_id="+tt.userID, bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*3))
			defer cancel()
			dsn := "postgres://tester:gidro4302@localhost:5433/testdb"
			pool, _ := repository.NewPool(ctx, dsn)

			storage := repository.NewUserPool(pool, kafkalogger.NewNop())

			service := service.New_Service(storage, kafkalogger.NewNop())

			h := handler.NewHandler(&kafkalogger.Logger{}, service)

			h.Create_Task(w, req)

			resp := w.Result()
			defer resp.Body.Close()
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectError {
				buf := new(bytes.Buffer)
				buf.ReadFrom(resp.Body)
				assert.Contains(t, buf.String(), tt.expectedBodyPart)
			} else if tt.expectedStatus == http.StatusCreated {
				var taskResp models.Task
				err := json.NewDecoder(resp.Body).Decode(&taskResp)
				assert.NoError(t, err)

				var sent struct {
					Title       string `json:"title"`
					Description string `json:"description"`
				}
				json.Unmarshal([]byte(tt.body), &sent)

				assert.Equal(t, sent.Title, taskResp.Title)
				assert.Equal(t, sent.Description, taskResp.Description)
			}

		})
	}

}
