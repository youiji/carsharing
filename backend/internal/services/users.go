package services

import (
	"context"
	"errors"
	"slices"
	"strings"

	"carsharing/internal/dto"
	"carsharing/internal/models"
	"carsharing/internal/repository"
	"carsharing/internal/util"
)

type UserService struct {
	users *repository.UserRepository
}

func NewUserService(users *repository.UserRepository) *UserService {
	return &UserService{users: users}
}

var errUserNotFound = util.ErrNotFound.WithMessage("Пользователь не найден")

func (s *UserService) List(ctx context.Context, f models.UserFilter) (dto.List[dto.User], error) {
	users, err := s.users.List(ctx, f)
	if err != nil {
		return dto.List[dto.User]{}, err
	}
	out := make([]dto.User, len(users))
	for i, u := range users {
		out[i] = dto.NewUser(u)
	}
	return dto.NewList(out), nil
}

func (s *UserService) ChangeRole(ctx context.Context, admin models.User, id int64, req dto.UserRoleRequest) (dto.User, error) {
	role := strings.TrimSpace(req.Role)
	if !slices.Contains(models.UserRoles, role) {
		return dto.User{}, util.ErrValidation.WithDetails(map[string]string{
			"role": "допустимые значения: " + strings.Join(models.UserRoles, ", "),
		})
	}
	if id == admin.ID {
		return dto.User{}, util.ErrCannotChangeOwnRole
	}
	return s.modify(ctx, id, func(u *models.User) { u.Role = role })
}

func (s *UserService) Update(ctx context.Context, id int64, req dto.UpdateUserRequest) (dto.User, error) {
	var v util.Validator
	if req.FullName == nil && req.DriverLicense == nil && req.Email == nil {
		v.Add("full_name", "передайте хотя бы одно из полей: full_name, driver_license, email")
	}
	var fullName, license, email string
	if req.FullName != nil {
		fullName = strings.TrimSpace(*req.FullName)
		v.StringLen("full_name", fullName, 2, 100)
	}
	if req.DriverLicense != nil {
		var ok bool
		if license, ok = util.NormalizeDriverLicense(*req.DriverLicense); !ok {
			v.Add("driver_license", "10 цифр, например 7712 345678")
		}
	}
	if req.Email != nil {
		email = strings.TrimSpace(*req.Email)
		v.Email("email", email)
	}
	if err := v.Err(); err != nil {
		return dto.User{}, err
	}

	return s.modify(ctx, id, func(u *models.User) {
		if req.FullName != nil {
			u.FullName = fullName
		}
		if req.DriverLicense != nil {
			u.DriverLicense = &license
		}
		if req.Email != nil {
			u.Email = email
		}
	})
}

func (s *UserService) Delete(ctx context.Context, admin models.User, id int64) error {
	if id == admin.ID {
		return util.ErrCannotDeleteSelf
	}
	err := s.users.Delete(ctx, id)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return errUserNotFound
	case errors.Is(err, util.ErrHasRelatedRecords):
		return util.ErrHasRelatedRecords.WithMessage("Нельзя удалить пользователя: у него есть аренды или записи обслуживания")
	}
	return err
}

// modify: заблокировать → применить change → если что-то изменилось и есть
// открытая аренда, отказать → сохранить. Всё в одной транзакции.
func (s *UserService) modify(ctx context.Context, id int64, change func(u *models.User)) (dto.User, error) {
	var saved models.User
	err := s.users.InTx(ctx, func(tx *repository.UserTx) error {
		u, err := tx.Lock(ctx, id)
		if errors.Is(err, repository.ErrNotFound) {
			return errUserNotFound
		}
		if err != nil {
			return err
		}
		before := u
		change(&u)
		if sameUserData(before, u) {
			saved = u // ничего не меняется — открытая аренда не мешает
			return nil
		}
		open, err := tx.HasOpenRental(ctx, id)
		if err != nil {
			return err
		}
		if open {
			return util.ErrUserHasActiveRental.WithMessage(
				"У пользователя есть бронирование или поездка: изменить роль и данные можно после её завершения")
		}
		saved, err = tx.Save(ctx, u)
		return err
	})
	if err != nil {
		return dto.User{}, err
	}
	return dto.NewUser(saved), nil
}

func sameUserData(a, b models.User) bool {
	same := a.Role == b.Role && a.FullName == b.FullName && strings.EqualFold(a.Email, b.Email)
	if a.DriverLicense == nil || b.DriverLicense == nil {
		return same && a.DriverLicense == nil && b.DriverLicense == nil
	}
	return same && *a.DriverLicense == *b.DriverLicense
}
