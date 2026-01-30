package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	handler "github.com/Explorerr/pet_project/internal/db-service/Handler"
	"github.com/Explorerr/pet_project/internal/db-service/testik"
	kafkalogger "github.com/Explorerr/pet_project/pkg/Kafka_logger"
	models "github.com/Explorerr/pet_project/pkg/Models"
	apperrors "github.com/Explorerr/pet_project/pkg/app_errors"
	"github.com/stretchr/testify/assert"
)

func Test_GetAllTasks(t *testing.T) {
	tests := []struct {
		Name             string
		expectedStatus   int
		expectError      bool
		Body             string
		expectedBodyPart string
		userID           string

		servic_mock testik.Service_mock
	}{
		{
			Name:           "Хороший тест",
			expectedStatus: 200,
			expectError:    false,
			userID:         "52522", expectedBodyPart: "",
			servic_mock: testik.Service_mock{
				Tasks: &[]models.Task{
					{
						ID:          1,
						UserID:      101,
						Title:       "Сделать домашнее задание",
						Description: "Написать тесты для HTTP-хендлера Login",
						Status:      " false",
						CreatedAt:   time.Now(),
					},
					{ID: 2,
						UserID:      102,
						Title:       "Купить продукты",
						Description: "Молоко, хлеб, яйца",
						Status:      "fals",
						CreatedAt:   time.Now(),
					},
				},
			}},
		{
			Name:             "Нету задач",
			expectedStatus:   404,
			expectError:      true,
			userID:           "52522",
			expectedBodyPart: "У вас неу задач",
			servic_mock: testik.Service_mock{
				Tasks: &[]models.Task{},
				Err:   apperrors.ErrEmptySlice,
			}},
		{
			Name:           "Непонятная ошибка",
			expectedStatus: 500,
			expectError:    false,
			userID:         "52522", expectedBodyPart: "Ошибка при получении задач",
			servic_mock: testik.Service_mock{
				Tasks: &[]models.Task{},
				Err:   errors.New("Ошибка непонятная"),
			}},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {

			req := httptest.NewRequest(http.MethodGet, "/?user_id="+tt.userID, nil)
			req.Header.Set("X-User-ID", tt.userID)

			w := httptest.NewRecorder()

			log := kafkalogger.Logger_For_Tests{}

			h := handler.NewHandler(&log, &tt.servic_mock)

			h.GetAllTasks(w, req)

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

			}
		})
	}

}
