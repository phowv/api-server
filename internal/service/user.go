package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"photo-viewer-server/internal/lib/mail"
	"photo-viewer-server/internal/storage/entity"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserExists = errors.New("user already exists")
	ErrUserPasswordTooShort = errors.New("password too short")
	ErrUserInvalidAuthentication = errors.New("invalid user authentication")
	ErrUserInvalidAuthorization = errors.New("invalid user authorization")
)

type UserRepo interface {
	CreateUser(ctx context.Context, user *entity.User) (uuid.UUID, error)
	GetUserByUuid(ctx context.Context, uuid uuid.UUID) (*entity.User, error)
	GetUserByEmail(ctx context.Context, email string) (*entity.User, error)
	GetUserByLogin(ctx context.Context, login string) (*entity.User, error)
	DeleteUser(ctx context.Context, uuid uuid.UUID) error
  UpdateUser(ctx context.Context, uuid uuid.UUID, fields map[string]any) error
}

type SessionRepo interface {
	SaveSession(ctx context.Context, refreshToken *entity.Session) (uuid.UUID, error)
	GetValidSessionByUuid(ctx context.Context, session uuid.UUID) (*entity.Session, error)
	RevokeSessionByUuid(ctx context.Context, sessionUuid uuid.UUID) error
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
	sessionRepo SessionRepo
	mailService *mail.MailService
}

type User struct {
	UserUuid uuid.UUID
	Role string
	Login string
	Email string
}

func NewUserService(log *slog.Logger, mailService *mail.MailService, userRepo UserRepo, sessionRepo SessionRepo) *UserService {
	return &UserService{
		log: log,
		userRepo: userRepo,
		sessionRepo: sessionRepo,
		mailService: mailService,
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

func hashToken(token string) (string, error) {
	digest := sha256.Sum256([]byte(token))
  bytes, err := bcrypt.GenerateFromPassword(digest[:], 14)

	return string(bytes), err
}

func compareTokens(token, hash string) bool {
	digest := sha256.Sum256([]byte(token))
	err := bcrypt.CompareHashAndPassword([]byte(hash), digest[:])
	return err == nil
}

func (s *UserService) CreateUser(ctx context.Context, data UserData) (uuid.UUID, error) {
	if len(data.Password) < 8 {
		return uuid.Nil, ErrUserPasswordTooShort
	}

	hashedPassword, err := hashPassword(data.Password)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := entity.User{
		Login: data.Login,
		Email: data.Email,
		Role: "user",
		Description: data.Description,
		HashPassword: hashedPassword,
	}

	existingUser, err := s.userRepo.GetUserByEmail(ctx, data.Email)

	if existingUser != nil {
		return uuid.Nil, ErrUserExists
	}

	existingUser, err = s.userRepo.GetUserByLogin(ctx, data.Login)

	if existingUser != nil {
		return uuid.Nil, ErrUserExists
	}

	id, err := s.userRepo.CreateUser(ctx, &user)

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create user: %w", err)
	}

	return id, nil
}

func (s *UserService) AuthenticateUser(ctx context.Context, userCredentials UserAuthCredentials) (*User, error) {
	if len(userCredentials.Password) < 8 {
		return nil, ErrUserInvalidAuthentication
	}

	user, err := s.userRepo.GetUserByLogin(ctx, userCredentials.Login)

	if err != nil {
		return nil, fmt.Errorf("error get user: %w", err)
	}

	if !comparePasswords(userCredentials.Password, user.HashPassword) {
		return nil, ErrUserInvalidAuthentication
	}

	return &User{
		UserUuid: user.UserUuid,
		Role: user.Role,
		Login: user.Login,
		Email: user.Email,
	}, nil
}

func (s *UserService) GetUserInfo(ctx context.Context, userUuid uuid.UUID) (*User, error) {
	user, err := s.userRepo.GetUserByUuid(ctx, userUuid)

	if err != nil {
		return nil, fmt.Errorf("error get user: %w", err)
	}

	return &User{
		UserUuid: user.UserUuid,
		Role: user.Role,
		Login: user.Login,
		Email: user.Email,
	}, nil
}

func (s *UserService) CreateSession(ctx context.Context, sessionUuid uuid.UUID, userUuid uuid.UUID, token string, expiresAt time.Time) error {
	hashToken, err := hashToken(token)

	if err != nil {
		return fmt.Errorf("failed to hash token: %w", err)
	}

	session := entity.Session{
		SessionUuid: sessionUuid,
		UserUuid: userUuid,
		ExpiresAt: expiresAt,
		HashToken: hashToken,
		IsRevoked: false,
	}

	_, err = s.sessionRepo.SaveSession(ctx, &session)

	if err != nil {
		return fmt.Errorf("error save session: %w", err)
	}

	return nil
}

func (s *UserService) AuthenticateSession(ctx context.Context, sessionUuid uuid.UUID, userUuid uuid.UUID, token string) (*User, error) {
	session, err := s.sessionRepo.GetValidSessionByUuid(ctx, sessionUuid)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if session.IsRevoked || !compareTokens(token, session.HashToken) {
		return nil, errors.New("expired token")
	}

	err = s.sessionRepo.RevokeSessionByUuid(ctx, sessionUuid)

	if err != nil {
		return nil, fmt.Errorf("failed to revoke session: %w", err)
	}

	if session.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("expired token")
	}

	return s.GetUserInfo(ctx, userUuid)
}
