package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/aouiniamine/aoui-drive/internal/database/sqlc"
	"github.com/aouiniamine/aoui-drive/internal/features/auth/dto"
	"github.com/aouiniamine/aoui-drive/internal/features/auth/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrClientInactive     = errors.New("client is inactive")
	ErrInvalidToken       = errors.New("invalid token")
)

type Claims struct {
	ClientID string `json:"client_id"`
	jwt.RegisteredClaims
}

type AuthService interface {
	Login(ctx context.Context, req dto.LoginRequest) (*dto.TokenResponse, error)
	ValidateToken(tokenString string) (*Claims, error)
	GetClientByID(ctx context.Context, id string) (*sqlc.Client, error)
	CreateClient(ctx context.Context, req dto.CreateClientRequest) (*dto.ClientResponse, error)
	RegenerateSecret(ctx context.Context, id string) (*dto.SecretResponse, error)
}

type authService struct {
	repo      repository.ClientRepository
	jwtSecret []byte
}

func New(repo repository.ClientRepository, jwtSecret string) AuthService {
	return &authService{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
	}
}

func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (*dto.TokenResponse, error) {
	client, err := s.repo.GetByAccessKey(ctx, req.AccessKey)
	if err != nil {
		if errors.Is(err, repository.ErrClientNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if client.IsActive == 0 {
		return nil, ErrClientInactive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(client.SecretKey), []byte(req.SecretKey)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.generateToken(client.ID)
}

func (s *authService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func (s *authService) GetClientByID(ctx context.Context, id string) (*sqlc.Client, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *authService) CreateClient(ctx context.Context, req dto.CreateClientRequest) (*dto.ClientResponse, error) {
	accessKey := generateAccessKey()
	secretKey := generateSecretKey()

	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(secretKey), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	client, err := s.repo.Create(ctx, sqlc.CreateClientParams{
		ID:        uuid.New().String(),
		Name:      req.Name,
		AccessKey: accessKey,
		SecretKey: string(hashedSecret),
		Role:      string(req.Role),
	})
	if err != nil {
		return nil, err
	}

	return &dto.ClientResponse{
		ID:        client.ID,
		Name:      client.Name,
		AccessKey: client.AccessKey,
		SecretKey: secretKey,
		Role:      dto.Role(client.Role),
	}, nil
}

func (s *authService) RegenerateSecret(ctx context.Context, id string) (*dto.SecretResponse, error) {
	secretKey := generateSecretKey()

	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(secretKey), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpdateSecret(ctx, id, string(hashedSecret)); err != nil {
		return nil, err
	}

	return &dto.SecretResponse{SecretKey: secretKey}, nil
}

func (s *authService) generateToken(clientID string) (*dto.TokenResponse, error) {
	expiry := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		ClientID: clientID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, err
	}

	return &dto.TokenResponse{
		AccessToken: tokenString,
		ExpiresIn:   int64(24 * 60 * 60),
	}, nil
}

func generateAccessKey() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return "AK" + hex.EncodeToString(bytes)
}

func generateSecretKey() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
