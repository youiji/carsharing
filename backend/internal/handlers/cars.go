package handlers

import (
	"net/http"

	"carsharing/internal/middleware"
	"carsharing/internal/models"
	"carsharing/internal/repository"
	"carsharing/internal/services"
	"carsharing/internal/util"
)

type CarHandler struct {
	svc *services.CarService
}

func (h *CarHandler) List(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.CurrentUser(r.Context())
	q := util.NewListQuery(r.URL.Query())
	f := models.CarFilter{
		Search:   q.Search(),
		TariffID: q.PositiveInt("tariff_id"),
	}
	if user.Role != models.RoleClient {
		f.Status = q.Enum("status", models.CarStatuses...)
	}
	f.Sort, f.Desc = q.Sort(repository.CarSortNames...)
	if err := q.Err(); err != nil {
		util.WriteError(w, r, err)
		return
	}
	list, err := h.svc.List(r.Context(), user, f)
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	util.WriteJSON(w, http.StatusOK, list)
}

func (h *CarHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := util.PathID(r, "id")
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	user, _ := middleware.CurrentUser(r.Context())
	car, err := h.svc.Get(r.Context(), user, id)
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	util.WriteJSON(w, http.StatusOK, car)
}
