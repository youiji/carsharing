package handlers

import (
	"context"
	"net/http"

	"carsharing/internal/dto"
	"carsharing/internal/middleware"
	"carsharing/internal/models"
	"carsharing/internal/services"
	"carsharing/internal/util"
)

type RentalHandler struct {
	svc *services.RentalService
}

func (h *RentalHandler) Book(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateRentalRequest
	if err := util.DecodeJSON(w, r, &req); err != nil {
		util.WriteError(w, r, err)
		return
	}
	user, _ := middleware.CurrentUser(r.Context())
	rental, err := h.svc.Book(r.Context(), user, req)
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	util.WriteJSON(w, http.StatusCreated, rental)
}

func (h *RentalHandler) Start(w http.ResponseWriter, r *http.Request) {
	h.action(w, r, h.svc.Start)
}

func (h *RentalHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	h.action(w, r, h.svc.Cancel)
}

func (h *RentalHandler) Finish(w http.ResponseWriter, r *http.Request) {
	id, err := util.PathID(r, "id")
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	var req dto.FinishRentalRequest
	if r.ContentLength != 0 {
		if err := util.DecodeJSON(w, r, &req); err != nil {
			util.WriteError(w, r, err)
			return
		}
	}
	user, _ := middleware.CurrentUser(r.Context())
	rental, err := h.svc.Finish(r.Context(), user, id, req)
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	util.WriteJSON(w, http.StatusOK, rental)
}

func (h *RentalHandler) action(w http.ResponseWriter, r *http.Request,
	fn func(context.Context, models.User, int64) (dto.Rental, error)) {
	id, err := util.PathID(r, "id")
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	user, _ := middleware.CurrentUser(r.Context())
	rental, err := fn(r.Context(), user, id)
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	util.WriteJSON(w, http.StatusOK, rental)
}
