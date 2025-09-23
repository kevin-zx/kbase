package token

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type TokenService interface {
	GetUserAccessToken() (*UserAccessToken, error)
	CreateUserAccessToken(token *UserAccessToken) error

	GetAuthCode() (*AuthCode, error)
	CreateAuthCode(code, state string) error

	Close() error
}

type tokenService struct {
	repo  TokenRepo
	token *UserAccessToken
}

func NewTokenService(db *gorm.DB) TokenService {
	repo := NewTokenRepo(db)
	return &tokenService{repo: repo}
}

func (s *tokenService) Close() error {
	// return s.repo.Close()
	return nil
}

var (
	ErrTokenExpired        = fmt.Errorf("token expired")
	ErrReFreshTokenExpired = fmt.Errorf("refresh token expired")
)

func (s *tokenService) GetUserAccessToken() (*UserAccessToken, error) {
	if s.token != nil && !s.token.IsExpired() && !s.token.IsRefreshTokenExpired() {
		return s.token, nil
	}

	token, err := s.repo.GetNewestUserAccessToken()
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if token.IsRefreshTokenExpired() {
		return nil, nil
	}
	if token.IsExpired() {
		return token, nil
	}
	return token, nil
}

func (s *tokenService) CreateUserAccessToken(token *UserAccessToken) error {
	return s.repo.CreateUserAccessToken(token)
}

var ErrAuthCodeExpired = fmt.Errorf("auth code expired")

func (s *tokenService) GetAuthCode() (*AuthCode, error) {
	ac, err := s.repo.GetNewestAuthCode()
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// 5min timeout
	if ac.CreatedAt.Add(5 * time.Minute).Before(time.Now()) {
		return nil, nil
	}
	return ac, nil
}

func (s *tokenService) CreateAuthCode(code, state string) error {
	return s.repo.CreateAuthCode(code, state)
}
