package auth

import (
	"fmt"
	"runtime"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func HashPassword(password string) (string, error) {
	CPU := uint8(1)
	CPUS := uint8(runtime.NumCPU())
	if CPUS > 1 {
		CPU = CPUS / 2
		if CPU < 1 {
			CPU = 1
		}
	}
	params := argon2id.Params{
		Memory:      976562,
		Iterations:  2,
		Parallelism: CPU,
		SaltLength:  20,
		KeyLength:   20,
	}
	hash, err := argon2id.CreateHash(password, &params)
	if err != nil {
		return "", fmt.Errorf("error hasing password: %v", err)
	}

	return hash, nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	ok, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, fmt.Errorf("error checking password: %v", err)
	}
	return ok, nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "chirpy-access",
		Subject:   userID.String(),
	}
	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	//key := hmac.New(sha256.New, []byte(tokenSecret))
	key := []byte(tokenSecret)
	signedstring, err := newToken.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("error creating signed string: %v", err)
	}

	return signedstring, nil

}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	},
		jwt.WithLeeway(time.Second*10))

	if err != nil {
		return uuid.Nil, err
	}

	subject, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.Nil, err
	}

	tokenUuid, err := uuid.Parse(subject)
	if err != nil {
		return uuid.Nil, err
	}

	return tokenUuid, nil

}
