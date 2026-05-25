package tokens

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
)

var (
	ErrUnknownClaimsType   = errors.New("unknown claims type")
	ErrJWTSignatureInvalid = errors.New("token signature invalid")
	ErrJWTExpired          = errors.New("token expired")
	ErrJWTNotValidYet      = errors.New("token not valid yer")
	ErrJWTInvalidKeyType   = errors.New("invalid key type")
	ErrJWTInvalidKey       = errors.New("invalid key")

	ErrCompareHashAndToken = errors.New("hash and token compare error")
)

type UserClaims struct {
	UID uint64
	jwt.RegisteredClaims
}

type TokensManager struct {
	JWTSecretKey []byte
	JWTDuration  time.Duration
	TokenLength  uint32
}

var once sync.Once
var instance TokensManager

func NewTokenManager(
	JWTSecretKey []byte,
	JWTDuration time.Duration,
	tokenLength uint32,
) (TokensManager, error) {
	var tmErr error
	once.Do(func() {
		instance = TokensManager{
			JWTSecretKey: JWTSecretKey,
			JWTDuration:  JWTDuration,
			TokenLength:  tokenLength,
		}
	})

	return instance, tmErr
}

func (tm TokensManager) NewJWT(uid uint64) (string, error) {
	claims := UserClaims{
		uid,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tm.JWTDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(tm.JWTSecretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (tm TokensManager) ParseJWT(tokenString string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&UserClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(tm.JWTSecretKey), nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
	)
	if err != nil {
		switch {
		// jwt/v5 errors -> local package errors
		case errors.Is(err, jwt.ErrTokenSignatureInvalid):
			err = ErrJWTSignatureInvalid
		case errors.Is(err, jwt.ErrTokenExpired):
			err = ErrJWTExpired
		case errors.Is(err, jwt.ErrTokenNotValidYet):
			err = ErrJWTNotValidYet
		case errors.Is(err, jwt.ErrInvalidKeyType):
			err = ErrJWTInvalidKeyType
		case errors.Is(err, jwt.ErrInvalidKey):
			err = ErrJWTInvalidKey
		}

		return nil, err
	}

	if claims, ok := token.Claims.(*UserClaims); ok {
		return claims, nil
	}
	return nil, ErrUnknownClaimsType
}

func (tm TokensManager) getRandBytes(length uint32) ([]byte, error) {
	bytes := make([]byte, length)

	_, err := rand.Read(bytes)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (tm TokensManager) NewToken() (string, error) {
	base, err := tm.getRandBytes(tm.TokenLength)
	if err != nil {
		return "", err
	}

	return base64.RawStdEncoding.EncodeToString(base), nil
}

func (tm TokensManager) HashTokenWithSalt(value, salt string) string {
	return base64.RawStdEncoding.EncodeToString(argon2.IDKey([]byte(value), []byte(salt), 3, 64*1024, 1, tm.TokenLength))
}

func (tm TokensManager) GetTokenHash(token string) (hash string, salt string, err error) {
	saltBytes, err := tm.getRandBytes(16)
	if err != nil {
		return "", "", err
	}

	salt = base64.RawStdEncoding.EncodeToString(saltBytes)
	hash = tm.HashTokenWithSalt(token, salt)

	return hash, salt, nil
}

func (tm TokensManager) CompareHashAndToken(token, hash, salt string) error {
	tokenHash := tm.HashTokenWithSalt(token, salt)

	if subtle.ConstantTimeCompare([]byte(tokenHash), []byte(hash)) == 0 {
		return ErrCompareHashAndToken
	}

	return nil
}

func (tm TokensManager) TokenExpired(expire time.Time) bool {
	return time.Now().After(expire)
}

// TODO: Добавить проверку на вход по старому токену
// TODO: Хранение старых токенов в отдельной таблице
