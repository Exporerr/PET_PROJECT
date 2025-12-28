package handlers_test

import (
	"bytes"

	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	handler "github.com/Explorerr/pet_project/internal/db-service/Handler"
	"github.com/Explorerr/pet_project/internal/db-service/testik"
	kafkalogger "github.com/Explorerr/pet_project/pkg/Kafka_logger"

	apperrors "github.com/Explorerr/pet_project/pkg/app_errors"
	"github.com/stretchr/testify/assert"
)

func Test_Delete(t *testing.T) {
	tests := []struct {
		Name             string
		Task_id          string
		User_id          string
		Expecetd_status  int
		expectedBodyPart string
		expectError      bool
		mock_service     testik.Service_mock
	}{
		{Name: "Хороший тест", Task_id: "25345252", User_id: "25345252", Expecetd_status: 200, expectedBodyPart: "", expectError: false, mock_service: testik.Service_mock{Err: nil}},
		{Name: "Не найдено задач", Task_id: "3455353", User_id: "45563346", Expecetd_status: 404, expectedBodyPart: "Задача не найдена", expectError: true, mock_service: testik.Service_mock{Err: apperrors.ErrTaskNotFound}},
		{Name: "Непонятно что либо упала транзакция", Task_id: "25345252", User_id: "25345252", Expecetd_status: 500, expectedBodyPart: "Ошибка", expectError: true, mock_service: testik.Service_mock{Err: errors.New("ошибка")}},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {

			req := httptest.NewRequest(http.MethodPatch, "/tasks/patch", nil)
			req.Header.Set("X-User-ID", tt.User_id)
			req.Header.Set("X-Task-ID", tt.Task_id)

			w := httptest.NewRecorder()

			log := kafkalogger.NewNop()

			h := handler.NewHandler(log, &tt.mock_service)

			h.Delete_Task(w, req)

			resp := w.Result()
			defer resp.Body.Close()
			assert.Equal(t, tt.Expecetd_status, resp.StatusCode)

			if tt.expectError {
				buf := new(bytes.Buffer)
				buf.ReadFrom(resp.Body)
				assert.Contains(t, buf.String(), tt.expectedBodyPart)
			}
		})
	}

}
