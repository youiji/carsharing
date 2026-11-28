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

type UserHandler struct {
	svc *services.UserService
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	q := util.NewListQuery(r.URL.Query())
	f := models.UserFilter{
		Search: q.Search(),
		Role:   q.Enum("role", models.UserRoles...),
	}
	f.Sort, f.Desc = q.Sort(repository.UserSortNames...)
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

func (h *UserHandler) ChangeRole(w http.ResponseWriter, r *http.Request) {
	id, err := util.PathID(r, "id")
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	var req dto.UserRoleRequest
	if err := util.DecodeJSON(w, r, &req); err != nil {
		util.WriteError(w, r, err)
		return
	}
	admin, _ := middleware.CurrentUser(r.Context())
	u, err := h.svc.ChangeRole(r.Context(), admin, id, req)
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	util.WriteJSON(w, http.StatusOK, u)
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := util.PathID(r, "id")
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	var req dto.UpdateUserRequest
	if err := util.DecodeJSON(w, r, &req); err != nil {
		util.WriteError(w, r, err)
		return
	}
	u, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	util.WriteJSON(w, http.StatusOK, u)
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := util.PathID(r, "id")
	if err != nil {
		util.WriteError(w, r, err)
		return
	}
	admin, _ := middleware.CurrentUser(r.Context())
	if err := h.svc.Delete(r.Context(), admin, id); err != nil {
		util.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
