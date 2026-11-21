package handlers

import (
	"net/http"

	"carsharing/internal/dto"
	"carsharing/internal/services"
	"carsharing/internal/util"
)

type AuthHandler struct {
	svc *services.AuthService
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := util.DecodeJSON(w, r, &req); err != nil {
		util.WriteError(w, r, err)
		return
	}
	user, err := h.svc.Register(r.Context(), req)
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	util.WriteJSON(w, http.StatusCreated, user)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var log dto.LoginRequest
	if err := util.DecodeJSON(w, r, &log); err != nil {
		util.WriteError(w, r, err)
		return
	}
	user, err := h.svc.Login(r.Context(), log)
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	util.WriteJSON(w, http.StatusOK, user)
}
