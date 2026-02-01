# адресс переменных окружения
ENV_API=internal/api-service/Configs/.env
ENV_DB=internal/db-service/configs/.env



# Подключаем, если файл существует
ifneq (,$(wildcard $(ENV_API)))
    include $(ENV_API)
    export
endif

ifneq (,$(wildcard $(ENV_DB)))
    include $(ENV_DB)
    export
endif
# DB_ENV, API_ENV — пути к .env файлам относительно корня проекта (при make cwd = корень).
# LOG_DIR — путь к папке логов относительно корня (cmd/db/logs или cmd/api/logs при make).
db-service-run:
	DB_ENV=internal/db-service/configs/.env LOG_DIR=cmd/db/logs go run cmd/db/main.go

api-service-run:
	API_ENV=internal/api-service/Configs/.env LOG_DIR=cmd/api/logs CONS_EVENTS=cmd/api/USER-EVENTS go run cmd/api/main.go

# Миграции базы данных
migrate-up:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" up


migrate-down:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" down

