package utils

import (
	"time"
	"go-media-center-example/internal/config"
	"github.com/golang-jwt/jwt/v4"
)

func GenerateToken(userID uint, cfg *config.Config) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWT.Secret))
}