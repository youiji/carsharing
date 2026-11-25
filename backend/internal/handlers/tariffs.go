package handlers

import (
	"net/http"

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
