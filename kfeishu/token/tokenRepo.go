package token

import (
	"time"

	"gorm.io/gorm"
)

type AuthCode struct {
	ID        int       `gorm:"primaryKey"`
	Code      string    `gorm:"not null"`
	State     string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"createdAt;default:CURRENT_TIMESTAMP"`
}

type UserAccessToken struct {
	ID                    int       `gorm:"primaryKey"`
	AccessToken           string    `gorm:"not null" json:"access_token"`
	ExpiresIn             int64     `gorm:"not null" json:"expires_in"`
	RefreshToken          string    `gorm:"refresh_token" json:"refresh_token"`
	RefreshTokenExpiresIn int64     `gorm:"refresh_token_expires_in" json:"refresh_token_expires_in"`
	Scope                 string    `gorm:"scope" json:"scope"`
	CreatedAt             time.Time `gorm:"createdAt;default:CURRENT_TIMESTAMP"`
}

func (UserAccessToken) TableName() string {
	return "user_access_token"
}

// is token expired
func (t *UserAccessToken) IsExpired() bool {
	return t.ExpiresIn+t.CreatedAt.Unix() < time.Now().Unix()
}

// is refresh token expired
func (t *UserAccessToken) IsRefreshTokenExpired() bool {
	return t.RefreshTokenExpiresIn+t.CreatedAt.Unix() < time.Now().Unix()
}

type TokenRepo interface {
	CreateAuthCode(code, state string) error
	GetNewestAuthCode() (*AuthCode, error)
	CreateUserAccessToken(token *UserAccessToken) error
	GetNewestUserAccessToken() (*UserAccessToken, error)
	Close() error
}

type tokenRepo struct {
	db *gorm.DB
}

func NewTokenRepo(db *gorm.DB) TokenRepo {
	// db.AutoMigrate(&AuthCode{}, &UserAccessToken{})
	return &tokenRepo{db: db}
}

func (r *tokenRepo) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (r *tokenRepo) CreateAuthCode(code, state string) error {
	authCode := &AuthCode{
		Code:  code,
		State: state,
	}
	return r.db.Create(authCode).Error
}

func (r *tokenRepo) GetNewestAuthCode() (*AuthCode, error) {
	authCode := &AuthCode{}
	err := r.db.Order("created_at desc").First(authCode).Error
	return authCode, err
}

func (r *tokenRepo) CreateUserAccessToken(token *UserAccessToken) error {
	return r.db.Create(token).Error
}

func (r *tokenRepo) GetNewestUserAccessToken() (*UserAccessToken, error) {
	token := &UserAccessToken{}
	err := r.db.Order("created_at desc").First(token).Error
	return token, err
}
