package handlers_test

import (
	"bytes"

	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	handler "github.com/Explorerr/pet_project/internal/db-service/Handler"
	"github.com/Explorerr/pet_project/internal/db-service/testik"
	kafkalogger "github.com/Explorerr/pet_project/pkg/Kafka_logger"

	apperrors "github.com/Explorerr/pet_project/pkg/app_errors"
	"github.com/stretchr/testify/assert"
)

func TestRegister(t *testing.T) {

	tests := []struct {
		json             string
		name             string
		mockService      *testik.Service_mock
		userID           string
		expectedStatus   int
		expectError      bool
		content_type     string
		body             string
		expectedBodyPart string
	}{
		{name: "Хороший тест",
			mockService:      &testik.Service_mock{ResultID: 1, Err: nil},
			body:             `{"username":"test","email":"test@email.com","password":"123456"}`,
			expectedStatus:   201,
			expectedBodyPart: "",
			content_type:     "application/json",
		},
		{name: "неверный тип контента",
			mockService:      &testik.Service_mock{ResultID: 1, Err: nil},
			expectedStatus:   415,
			body:             `{"username":"test","email":"test@email.com","password":"123456"}`,
			expectedBodyPart: "Content-Type должен быть application/json",
			content_type:     "text/plain",
		},
		{name: "уже существует",
			mockService: &testik.Service_mock{ResultID: 1, Err: apperrors.ErrUserAlreadyExists},
			body:        `{"username":"test","email":"test@email.com","password":"123456"}`, expectedStatus: 409,
			expectedBodyPart: "Такой пользователь уже сущетвует ",
			content_type:     "application/json",
		},
		{name: "Unsupported type",
			mockService:      &testik.Service_mock{ResultID: 1, Err: nil},
			body:             `{invalid json`,
			expectedStatus:   400,
			expectedBodyPart: "неверный формат JSON",
			content_type:     "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", tt.content_type)
			w := httptest.NewRecorder()

			h := handler.NewHandler(&kafkalogger.Logger_For_Tests{}, tt.mockService)

			h.POST_USER(w, req)

			resp := w.Result()
			defer resp.Body.Close()
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			if tt.expectedBodyPart != "" {
				buf := new(bytes.Buffer)
				buf.ReadFrom(resp.Body)
				body := buf.String()

				if !strings.Contains(body, tt.expectedBodyPart) {
					t.Errorf("Ожидали сообщение: %q, получили: %s", tt.expectedBodyPart, body)
				}

			}

		})
	}

}
