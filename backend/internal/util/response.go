package util

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
)

const maxBodyBytes = 1 << 20 // 1 МБ

type errorBody struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		log.Printf("unhandled error %s %s: %v", r.Method, r.URL.Path, err)
		appErr = ErrInternal
	}
	WriteJSON(w, appErr.Status, errorBody{Error: errorPayload{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
	}})
}

func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)

	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return ErrValidation.WithMessage("Пустое тело запроса")
		}
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return ErrValidation.WithMessage("Слишком большое тело запроса")
		}
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) && typeErr.Field != "" {
			return ErrValidation.WithDetails(map[string]string{typeErr.Field: "неверный тип значения"})
		}
		return ErrValidation.WithMessage("Некорректный JSON в теле запроса")
	}
	if dec.More() {
		return ErrValidation.WithMessage("Некорректный JSON в теле запроса")
	}
	return nil
}
