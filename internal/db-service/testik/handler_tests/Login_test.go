package handlers

import (
	"bytes"
	"encoding/json"

	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	handler "github.com/Explorerr/pet_project/internal/db-service/Handler"
	"github.com/Explorerr/pet_project/internal/db-service/testik"
	"github.com/stretchr/testify/assert"

	kafkalogger "github.com/Explorerr/pet_project/pkg/Kafka_logger"
	models "github.com/Explorerr/pet_project/pkg/Models"
	apperrors "github.com/Explorerr/pet_project/pkg/app_errors"
)

func Test_Login(t *testing.T) {
	tests := []struct {
		Name             string
		expectedStatus   int
		expectError      bool
		ContentType      string
		Body             string
		expectedBodyPart string

		servic_mock testik.Service_mock
	}{
		{
			Name:             "Хороший тест",
			expectedStatus:   200,
			expectError:      false,
			ContentType:      "application/json",
			expectedBodyPart: "",
			Body:             `{"email":"test@htomail.com","password":"34552"}`,
			servic_mock:      testik.Service_mock{Err: nil, User: &models.User{Email: "HTOMAIL.COM"}},
		},
		{
			Name:             "Неверный Content-Type",
			expectedStatus:   415,
			expectError:      true,
			Body:             `{"email":"test@htomail.com","password":"34552"}`,
			ContentType:      "text/plain",
			expectedBodyPart: "Content-Type должен быть application/json",
			servic_mock:      testik.Service_mock{Err: nil, User: &models.User{Email: "HTOMAIL.COM"}},
		},

		{
			Name:             "Broken JSON ",
			expectedStatus:   400,
			expectError:      true,
			ContentType:      "application/json",
			Body:             `{invalid json`,
			expectedBodyPart: "неверный формат JSON",
			servic_mock:      testik.Service_mock{Err: nil, User: &models.User{Email: "HTOMAIL.COM"}},
		},
		{
			Name:             "Пользователя не существует",
			expectedStatus:   404,
			expectError:      true,
			ContentType:      "application/json",
			expectedBodyPart: "Такой пользователь не сущетвует ",
			Body:             `{"email":"test@htomail.com","password":"34552"}`,
			servic_mock:      testik.Service_mock{Err: apperrors.ErrUserNotExist, User: &models.User{Email: "HTOMAIL.COM"}},
		}, {
			Name:             "неизветная ошибка",
			expectedStatus:   500,
			expectError:      true,
			ContentType:      "application/json",
			expectedBodyPart: "internal error",
			Body:             `{"email":"test@htomail.com","password":"34552"}`,
			servic_mock:      testik.Service_mock{Err: errors.New("непонятно"), User: &models.User{Email: "HTOMAIL.COM"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {

			req := httptest.NewRequest(http.MethodGet, "/tasks/login", bytes.NewReader([]byte(tt.Body)))
			req.Header.Set("Content-Type", tt.ContentType)
			w := httptest.NewRecorder()
			log := kafkalogger.NewNop()

			h := handler.NewHandler(log, &tt.servic_mock)

			h.Login(w, req)

			resp := w.Result()
			defer resp.Body.Close()
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectError {
				buf := new(bytes.Buffer)
				buf.ReadFrom(resp.Body)
				assert.Contains(t, buf.String(), tt.expectedBodyPart)
			} else if tt.expectedStatus == http.StatusOK {
				var taskResp models.User
				err := json.NewDecoder(resp.Body).Decode(&taskResp)
				assert.NoError(t, err)

			}

		})
	}
}
