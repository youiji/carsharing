package handlers

import (
	"net/http"

	"carsharing/internal/dto"
	"carsharing/internal/middleware"
	"carsharing/internal/models"
	"carsharing/internal/repository"
	"carsharing/internal/services"
	"carsharing/internal/util"
)

type MaintenanceHandler struct {
	svc *services.MaintenanceService
}

func (h *MaintenanceHandler) List(w http.ResponseWriter, r *http.Request) {
	q := util.NewListQuery(r.URL.Query())
	f := models.MaintenanceFilter{
		CarID:  q.PositiveInt("car_id"),
		Search: q.Search(),
	}
	f.Sort, f.Desc = q.Sort(repository.MaintenanceSortNames...)
	if err := q.Err(); err != nil {
		util.WriteError(w, r, err)
		return
	}
	list, err := h.svc.List(r.Context(), f)
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	util.WriteJSON(w, http.StatusOK, list)
}

func (h *MaintenanceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateMaintenanceRequest
	if err := util.DecodeJSON(w, r, &req); err != nil {
		util.WriteError(w, r, err)
		return
	}
	user, _ := middleware.CurrentUser(r.Context())
	m, err := h.svc.Create(r.Context(), user, req)
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	util.WriteJSON(w, http.StatusCreated, m)
}

func (h *MaintenanceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := util.PathID(r, "id")
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	var req dto.UpdateMaintenanceRequest
	if err := util.DecodeJSON(w, r, &req); err != nil {
		util.WriteError(w, r, err)
		return
	}
	user, _ := middleware.CurrentUser(r.Context())
	m, err := h.svc.Update(r.Context(), user, id, req)
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	util.WriteJSON(w, http.StatusOK, m)
}

func (h *MaintenanceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := util.PathID(r, "id")
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	user, _ := middleware.CurrentUser(r.Context())
	if err := h.svc.Delete(r.Context(), user, id); err != nil {
		util.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
