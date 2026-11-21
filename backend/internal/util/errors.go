package util

import (
	"fmt"
	"net/http"
)

type AppError struct {
	Status  int
	Code    string
	Message string
	Details map[string]string
}

func (e *AppError) Error() string {
	return fmt.Sprintf("%d %s: %s", e.Status, e.Code, e.Message)
}

func (e *AppError) WithDetails(details map[string]string) *AppError {
	c := *e
	c.Details = details
	return &c
}

func (e *AppError) WithMessage(msg string) *AppError {
	c := *e
	c.Message = msg
	return &c
}

func newErr(status int, code, msg string) *AppError {
	return &AppError{Status: status, Code: code, Message: msg}
}

var (
	ErrValidation         = newErr(http.StatusBadRequest, "validation_error", "Некорректные данные запроса")
	ErrUnauthorized       = newErr(http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
	ErrForbidden          = newErr(http.StatusForbidden, "forbidden", "Недостаточно прав")
	ErrNotFound           = newErr(http.StatusNotFound, "not_found", "Объект не найден")
	ErrInternal           = newErr(http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
	ErrEmailTaken         = newErr(http.StatusConflict, "email_taken", "Email уже занят")
	ErrLicenseTaken       = newErr(http.StatusConflict, "license_taken", "Водительское удостоверение уже зарегистрировано")
	ErrInvalidCredentials = newErr(http.StatusUnauthorized, "invalid_credentials", "Неверный email или пароль")
	ErrWrongPassword      = newErr(http.StatusBadRequest, "wrong_password", "Неверный текущий пароль при смене пароля")
)
