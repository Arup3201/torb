package auth

import (
	"testing"
	"time"

	"github.com/Arup3201/torb/core"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestNewClaims(t *testing.T) {
	exp := time.Now().Add(1 * time.Hour)

	claims := NewClaims("iss", "sub", exp)

	assert.Equal(t, "iss", claims.Issuer)
	assert.Equal(t, "sub", claims.Subject)
	assert.Equal(t, exp, claims.Expiry)
	assert.NotNil(t, claims.IssuedAt)
	assert.Equal(t, claims.IssuedAt, claims.NotBefore)
}

func TestJWTFromClaims_AlgRSA(t *testing.T) {
	exp := time.Now().Add(1 * time.Hour)
	claims := NewClaims("iss", "sub", exp)

	tokenObj, err := JWTFromClaims(claims, JWT_ALG_RSA)

	assert.NoError(t, err)
	assert.NotNil(t, tokenObj)
	assert.Equal(t, tokenObj.Alg, jwt.SigningMethodRS256)
	assert.Equal(t, "iss", tokenObj.Claims.Issuer)
	assert.Equal(t, "sub", tokenObj.Claims.Subject)
	assert.Equal(t, exp, tokenObj.Claims.Expiry)
}

func TestJWTFromClaims_AlgInvalid(t *testing.T) {
	exp := time.Now().Add(1 * time.Hour)
	claims := NewClaims("iss", "sub", exp)

	_, err := JWTFromClaims(claims, "None")

	assert.ErrorIs(t, err, core.ErrInvalidValue)
}

func TestJWTSign(t *testing.T) {
	exp := time.Now().Add(1 * time.Hour)
	claims := NewClaims("iss", "sub", exp)
	tokenObj, _ := JWTFromClaims(claims, JWT_ALG_HMAC)
	key := []byte("a-secret-key")

	token, err := tokenObj.Sign(key)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestJWTSign_VerifySuccess(t *testing.T) {
	exp := time.Now().Add(1 * time.Hour)
	claims := NewClaims("iss", "sub", exp)
	tokenObj, _ := JWTFromClaims(claims, JWT_ALG_HMAC)
	key := []byte("a-secret-key")

	tokenString, err := tokenObj.Sign(key)

	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (any, error) {
		return key, nil
	})
	assert.NoError(t, err)
	tokenClaims, ok := token.Claims.(*CustomClaims)
	assert.True(t, ok)
	assert.Equal(t, claims.Jti, tokenClaims.Jti)
	assert.Equal(t, "iss", tokenClaims.Issuer)
	assert.Equal(t, "sub", tokenClaims.Subject)
}

func TestJWTSign_VerifyFailWrongKey(t *testing.T) {
	exp := time.Now().Add(1 * time.Hour)
	claims := NewClaims("iss", "sub", exp)
	tokenObj, _ := JWTFromClaims(claims, JWT_ALG_HMAC)
	key := []byte("a-secret-key")

	tokenString, err := tokenObj.Sign(key)

	_, err = jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (any, error) {
		return []byte("wrong"), nil
	})
	assert.Error(t, err)
}

func TestJWTSign_VerifyFailExpiredToken(t *testing.T) {
	exp := time.Now().Add(-1 * time.Hour)
	claims := NewClaims("iss", "sub", exp)
	tokenObj, _ := JWTFromClaims(claims, JWT_ALG_HMAC)
	key := []byte("a-secret-key")

	tokenString, err := tokenObj.Sign(key)

	_, err = jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (any, error) {
		return key, nil
	})
	assert.Error(t, err)
}

func TestClaimsFromToken(t *testing.T) {
	exp := time.Now().Add(1 * time.Hour)
	claims := NewClaims("iss", "sub", exp)
	tokenObj, _ := JWTFromClaims(claims, JWT_ALG_HMAC)
	key := []byte("a-secret-key")
	tokenString, _ := tokenObj.Sign(key)

	claims, err := ClaimsFromToken(tokenString, key)

	assert.NoError(t, err)
	assert.Equal(t, "iss", claims.Issuer)
	assert.Equal(t, "sub", claims.Subject)
}

func TestClaimsFromToken_FailWrongKey(t *testing.T) {
	exp := time.Now().Add(1 * time.Hour)
	claims := NewClaims("iss", "sub", exp)
	tokenObj, _ := JWTFromClaims(claims, JWT_ALG_HMAC)
	key := []byte("a-secret-key")
	tokenString, _ := tokenObj.Sign(key)

	claims, err := ClaimsFromToken(tokenString, []byte("wrong"))

	assert.Error(t, err)
}

func TestClaimsFromToken_FailAlgorithmNone(t *testing.T) {
	exp := time.Now().Add(1 * time.Hour)
	now := time.Now()
	claims := CustomClaims{
		Jti: "123",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "iss",
			Subject:   "sub",
		},
	}
	key := []byte("a-secret-key")
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	unsignedString, _ := token.SigningString()

	_, err := ClaimsFromToken(unsignedString, key)

	assert.Error(t, err)
}
