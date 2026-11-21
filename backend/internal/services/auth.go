package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"carsharing/internal/dto"
	"carsharing/internal/models"
	"carsharing/internal/repository"
	"carsharing/internal/util"
)

type UserStore interface {
	Create(ctx context.Context, u models.User) (models.User, error)
	GetByEmail(ctx context.Context, email string) (models.User, error)
	GetByID(ctx context.Context, id int64) (models.User, error)
	UpdatePasswordHash(ctx context.Context, id int64, hash string) error
}

type AuthService struct {
	users  UserStore
	tokens *util.TokenManager
}

func NewAuthService(users UserStore, tokens *util.TokenManager) *AuthService {
	return &AuthService{users: users, tokens: tokens}
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

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (dto.LoginResponse, error) {
	email := strings.TrimSpace(req.Email)

	var v util.Validator
	if email == "" {
		v.Add("email", "обязательное поле")
	}
	if req.Password == "" {
		v.Add("password", "обязательное поле")
	}
	if err := v.Err(); err != nil {
		return dto.LoginResponse{}, err
	}

	user, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, repository.ErrNotFound) {
		util.SpendPasswordCheck(req.Password)
		return dto.LoginResponse{}, util.ErrInvalidCredentials
	}
	if err != nil {
		return dto.LoginResponse{}, err
	}

	ok, err := util.CheckPassword(user.PasswordHash, req.Password)
	if err != nil {
		return dto.LoginResponse{}, err
	}
	if !ok {
		return dto.LoginResponse{}, util.ErrInvalidCredentials
	}

	token, err := s.tokens.Issue(user.ID, user.Role)
	if err != nil {
		return dto.LoginResponse{}, err
	}
	return dto.LoginResponse{
		Token:     token,
		TokenType: "Bearer",
		ExpiresIn: int(s.tokens.TTL().Seconds()),
		User:      dto.NewUser(user),
	}, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID int64, req dto.ChangePasswordRequest) error {
	var v util.Validator
	if req.CurrentPassword == "" {
		v.Add("current_password", "обязательное поле")
	}
	v.Password("new_password", req.NewPassword)
	if err := v.Err(); err != nil {
		return err
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("load user: %w", err)
	}
	ok, err := util.CheckPassword(user.PasswordHash, req.CurrentPassword)
	if err != nil {
		return err
	}
	if !ok {
		return util.ErrWrongPassword
	}

	hash, err := util.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	return s.users.UpdatePasswordHash(ctx, userID, hash)
}
