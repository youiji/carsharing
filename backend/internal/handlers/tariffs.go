package handlers

import (
	"net/http"

	"carsharing/internal/dto"
	"carsharing/internal/services"
	"carsharing/internal/util"
)

type TariffHandler struct {
	svc *services.TariffService
}

func (h *TariffHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context())
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	util.WriteJSON(w, http.StatusOK, list)
}

func (h *TariffHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.TariffRequest
	if err := util.DecodeJSON(w, r, &req); err != nil {
		util.WriteError(w, r, err)
		return
	}
	tariff, err := h.svc.Create(r.Context(), req)
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	util.WriteJSON(w, http.StatusCreated, tariff)
}

func (h *TariffHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := util.PathID(r, "id")
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	var req dto.TariffRequest
	if err := util.DecodeJSON(w, r, &req); err != nil {
		util.WriteError(w, r, err)
		return
	}
	tariff, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	util.WriteJSON(w, http.StatusOK, tariff)
}

func (h *TariffHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := util.PathID(r, "id")
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		util.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
