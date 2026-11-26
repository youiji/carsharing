package handlers

import (
	"net/http"

	"carsharing/internal/dto"
	"carsharing/internal/middleware"
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
