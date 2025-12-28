package models

type Request_Task struct { //ЗАПРОС ОТ КЛИЕНТА (СОЗДАНИЕ ЗАДАЧИ)
	Title       string `json:"title"`
	Description string `json:"description"`
}

type Request_Register struct { // ЗАПРОС ОТ КЛИЕНТА (РЕГИСТРАЦИЯ)
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Request_Login struct { // ЗАПРОС ОТ КЛИЕНТА (ЛОГИН)
	Email    string `json:"email"`
	Password string `json:"password"`
}
