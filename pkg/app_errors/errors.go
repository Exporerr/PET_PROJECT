package apperrors

import (
	"errors"
)

var ErrEmptySlice = errors.New("у вас нету задач")
var ErrInvalidInput = errors.New("неверный ввод")
var ErrUserNotExist = errors.New("пользователя не существует")
var ErrUserAlreadyExists = errors.New("пользователь уже существует")
var ErrWentWrong = errors.New("упс! Что-то пошло не так")
var ErrTransactionNotInit = errors.New(" Ошибка инициализации транзакции")
var ErrTaskNotFound = errors.New("такой задачи нет ")
