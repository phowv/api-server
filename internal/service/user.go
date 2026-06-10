package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"photo-viewer-server/internal/lib/mail"
	"photo-viewer-server/internal/storage/entity"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserExists = errors.New("user already exists")
	ErrUserPasswordTooShort = errors.New("password too short")
	ErrUserInvalidAuthentication = errors.New("invalid user authentication")
	ErrUserInvalidAuthorization = errors.New("invalid user authorization")
)

type UserRepo interface {
	CreateUser(ctx context.Context, user *entity.User) (int, error)
	GetUserById(ctx context.Context, id int) (*entity.User, error)
	GetUserByEmail(ctx context.Context, email string) (*entity.User, error)
	GetUserByLogin(ctx context.Context, login string) (*entity.User, error)
	DeleteUser(ctx context.Context, id int) error
  UpdateUser(ctx context.Context, id int, fields map[string]any) error
}

type UserData struct {
	Login string `json:"login"`
	Email string `json:"email"`
	Password string `json:"password"`
	Description string `json:"description"`
}

type UserAuthCredentials struct {
	Login string `json:"login"`
	Password string `json:"password"`
}

type UserService struct {
	log *slog.Logger
	userRepo UserRepo
	MailService *mail.MailService
}

type User struct {
	UserId int
	Role string
	Login string
	Email string
}

func NewUserService(log *slog.Logger, mailService *mail.MailService, userRepo UserRepo) *UserService {
	return &UserService{
		log: log,
		userRepo: userRepo,
		MailService: mailService,
	}
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func comparePasswords(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (u *UserService) CreateUser(ctx context.Context, data UserData) (int, error) {
	if len(data.Password) < 8 {
		return 0, ErrUserPasswordTooShort
	}

	hashedPassword, err := hashPassword(data.Password)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %w", err)
	}

	user := entity.User{
		Login: data.Login,
		Email: data.Email,
		Role: "user",
		Description: data.Description,
		HashPassword: hashedPassword,
	}

	existingUser, err := u.userRepo.GetUserByEmail(ctx, data.Email)

	if existingUser != nil {
		return 0, ErrUserExists
	}

	existingUser, err = u.userRepo.GetUserByLogin(ctx, data.Login)

	if existingUser != nil {
		return 0, ErrUserExists
	}

	id, err := u.userRepo.CreateUser(ctx, &user)

	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}

	return id, nil
}

func (u *UserService) AuthenticateUser(ctx context.Context, userCredentials UserAuthCredentials) (*User, error) {
	if len(userCredentials.Password) < 8 {
		return nil, ErrUserInvalidAuthentication
	}

	user, err := u.userRepo.GetUserByLogin(ctx, userCredentials.Login)

	if err != nil {
		return nil, fmt.Errorf("error get user: %w", err)
	}

	if !comparePasswords(userCredentials.Password, user.HashPassword) {
		return nil, ErrUserInvalidAuthentication
	}

	return &User{
		UserId: user.UserId,
		Role: user.Role,
		Login: user.Login,
		Email: user.Email,
	}, nil
}
