package services

import (
	"context"
	"strings"

	"carsharing/internal/dto"
	"carsharing/internal/models"
	"carsharing/internal/util"
)

type UserStore interface {
	Create(ctx context.Context, u models.User) (models.User, error)
}

type AuthService struct {
	users UserStore
}

func NewAuthService(users UserStore) *AuthService {
	return &AuthService{users: users}
}

func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (dto.User, error) {
	email := strings.TrimSpace(req.Email)
	fullName := strings.TrimSpace(req.FullName)

	var v util.Validator
	v.Email("email", email)
	v.Password("password", req.Password)
	v.StringLen("full_name", fullName, 2, 100)
	license, ok := util.NormalizeDriverLicense(req.DriverLicense)
	switch {
	case strings.TrimSpace(req.DriverLicense) == "":
		v.Add("driver_license", "обязательное поле")
	case !ok:
		v.Add("driver_license", "10 цифр, например 7712 345678")
	}
	if err := v.Err(); err != nil {
		return dto.User{}, err
	}

	hash, err := util.HashPassword(req.Password)
	if err != nil {
		return dto.User{}, err
	}
	created, err := s.users.Create(ctx, models.User{
		Email:         email,
		PasswordHash:  hash,
		FullName:      fullName,
		DriverLicense: &license,
		Role:          models.RoleClient,
	})
	if err != nil {
		return dto.User{}, err
	}
	return dto.NewUser(created), nil
}
