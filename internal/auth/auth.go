package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("unable to generate hash for password: %s - %v", password, err)
		return "", err
	}

	return string(hashedPassword), nil
}

func CheckPasswordHash(hash, password string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return false, err
	}

	return true, nil
}

const MY_SECRET_KEY = "AllYourBase"

type MyCustomClaims struct {
	jwt.RegisteredClaims
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	mySigningKey := []byte(MY_SECRET_KEY)
	claims := MyCustomClaims{
		jwt.RegisteredClaims{
			// A usual scenario is to set the expiration time relative to the current time
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			NotBefore: jwt.NewNumericDate(time.Now().UTC()),
			Issuer:    "chirpy",
			Subject:   userID.String(),
			ID:        "1",
			Audience:  []string{userID.String()},
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString(mySigningKey)
	if err != nil {
		log.Fatal("unable to sign token", err)
		return "", err
	}

	return ss, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &MyCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(MY_SECRET_KEY), nil
	})
	if err != nil {
		fmt.Printf("unable to parse token: %v\n", err)
		return uuid.Nil, err
	}

	if claims, ok := token.Claims.(*MyCustomClaims); ok {
		return uuid.MustParse(claims.Subject), nil
	} else {
		log.Fatal("unknown claims type, cannot proceed")
		return uuid.UUID{}, errors.New("Unable to get claims from token")
	}
}

func GetBearerToken(headers http.Header) (string, error) {
	authorization := headers.Get("Authorization")
	if authorization != "" {
		parts := strings.Split(authorization, "Bearer ")
		token := parts[1]
		return token, nil
	} else {
		return "", errors.New("no authorizzation header provided")
	}
}

func MakeRefreshToken() (string, error) {
	key := make([]byte, 32)
	n, err := rand.Read(key)
	if err != nil {
		fmt.Printf("unable to read bytes - err: %v\n", err)
		return "", err
	} else if n == 0 {
		fmt.Println("unable to read bytes - zero length")
		return "", errors.New("unable to read bytes - zero length")
	}
	return hex.EncodeToString(key), nil
}
