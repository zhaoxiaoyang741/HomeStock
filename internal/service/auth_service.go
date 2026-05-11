package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
)

var (
	ErrUserExists   = errors.New("username already exists")
	ErrInvalidCreds = errors.New("invalid username or password")
	ErrInvalidToken = errors.New("invalid or expired token")
)

// AuthClaims is the JWT claims payload carried in every access token.
type AuthClaims struct {
	UserID      uint   `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	jwt.RegisteredClaims
}

// AuthService provides user registration, login, and JWT token operations.
type AuthService struct {
	db        *gorm.DB
	jwtSecret []byte
	tokenTTL  time.Duration
}

// NewAuthService creates an AuthService. If jwtSecret is empty, a random
// 32-byte key is generated. Use GetSecretHex() to retrieve the generated key.
func NewAuthService(db *gorm.DB, jwtSecret string, tokenDurationMinutes int) *AuthService {
	secret := []byte(jwtSecret)
	if len(secret) == 0 {
		key := make([]byte, 32)
		_, _ = rand.Read(key)
		secret = key
	}
	ttl := time.Duration(tokenDurationMinutes) * time.Minute
	if ttl <= 0 {
		ttl = 1440 * time.Minute
	}
	return &AuthService{
		db:        db,
		jwtSecret: secret,
		tokenTTL:  ttl,
	}
}

// GetSecretHex returns the JWT signing key as a hex string, useful when a
// random key was generated at startup and needs to be persisted.
func (s *AuthService) GetSecretHex() string {
	return hex.EncodeToString(s.jwtSecret)
}

// Register creates a new user after validating input and checking uniqueness.
// The password is hashed with bcrypt before storage.
func (s *AuthService) Register(ctx context.Context, username, password, displayName string) (*model.User, error) {
	username = strings.TrimSpace(username)
	if username == "" || len(username) < 3 {
		return nil, errors.New("username must be at least 3 characters")
	}
	if len(password) < 6 {
		return nil, errors.New("password must be at least 6 characters")
	}

	var count int64
	if err := s.db.WithContext(ctx).Model(&model.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return nil, fmt.Errorf("check username: %w", err)
	}
	if count > 0 {
		return nil, ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	d := strings.TrimSpace(displayName)
	if d == "" {
		d = username
	}

	user := &model.User{
		Username:     username,
		PasswordHash: string(hash),
		DisplayName:  d,
	}
	if err := s.db.WithContext(ctx).Create(user).Error; err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

// Login verifies credentials and returns a signed JWT token with the user.
func (s *AuthService) Login(ctx context.Context, username, password string) (token string, user *model.User, err error) {
	var u model.User
	if err := s.db.WithContext(ctx).Where("username = ?", strings.TrimSpace(username)).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, ErrInvalidCreds
		}
		return "", nil, fmt.Errorf("find user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", nil, ErrInvalidCreds
	}

	now := time.Now()
	claims := AuthClaims{
		UserID:      u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   fmt.Sprintf("%d", u.ID),
		},
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := jwtToken.SignedString(s.jwtSecret)
	if err != nil {
		return "", nil, fmt.Errorf("sign token: %w", err)
	}
	return tokenStr, &u, nil
}

// ValidateToken parses and validates a JWT token string, returning the claims
// on success.
func (s *AuthService) ValidateToken(tokenString string) (*AuthClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}
	claims, ok := token.Claims.(*AuthClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// GetUserByID returns the user with the given primary key.
func (s *AuthService) GetUserByID(ctx context.Context, id uint) (*model.User, error) {
	var u model.User
	if err := s.db.WithContext(ctx).First(&u, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}
