package auth

import (
	"errors"
	"time"

	"warehousecontrol/internal/models"

	"github.com/golang-jwt/jwt/v5"
)

type JWT struct {
	secret []byte
}

func New(secret string) *JWT {
	return &JWT{secret: []byte(secret)}
}

type tokenClaims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func (j *JWT) Issue(username, role string) (string, error) {
	claims := tokenClaims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secret)
}

func (j *JWT) Parse(tokenStr string) (*models.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &tokenClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return j.secret, nil
	})
	if err != nil {
		return nil, models.ErrInvalidToken
	}
	claims, ok := token.Claims.(*tokenClaims)
	if !ok || !token.Valid {
		return nil, models.ErrInvalidToken
	}
	return &models.Claims{
		Username: claims.Username,
		Role:     claims.Role,
	}, nil
}
