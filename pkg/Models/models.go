package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Request_Task struct { //ЗАПРОС ОТ КЛИЕНТА (СОЗДАНИЕ ЗАДАЧИ) ОТ API СЕРВИСА....
	Title       string `json:"title"`
	Description string `json:"description"`
}
type Request_api_Task struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type Request_Register struct { // ЗАПРОС ОТ КЛИЕНТА (РЕГИСТРАЦИЯ)
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	IP_Adres string
}

type Request_Login struct { // ЗАПРОС ОТ КЛИЕНТА (ЛОГИН)
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Responser_Register struct {
	User_ID    int       `json:"user_id"`
	Username   string    `json:"username"`
	Created_at time.Time `json:"created_at"`
}
type User struct {
	ID            int       `json:"id"`
	Username      string    `json:"username"`
	Password_Hash string    `json:"password_hash"` // пароль или хэш лучше не возвращать наружу
	Email         string    `json:"email"`
	CreatedAt     time.Time `json:"created_at"`
	Role          string    `json:"role"`
}

// задачи пользователя
type Task struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	Title       string    `json:"title"`                 // заполняем
	Description string    `json:"description,omitempty"` // заполняем
	Status      bool      `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}
type MyClaims struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}
