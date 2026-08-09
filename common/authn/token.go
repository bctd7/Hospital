package authn

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var (
	ErrInvalidTokenConfig = errors.New("invalid token configuration")
	ErrInvalidToken       = errors.New("invalid access token")
	ErrInactiveAccount    = errors.New("account is not active")
	ErrSigningUnavailable = errors.New("access token signing key is unavailable")
)

type TokenConfig struct {
	Issuer          string
	Audience        string
	SigningKey      ed25519.PrivateKey
	VerificationKey ed25519.PublicKey
	TTL             time.Duration
}

type TokenManager struct {
	issuer          string
	audience        string
	signingKey      ed25519.PrivateKey
	verificationKey ed25519.PublicKey
	ttl             time.Duration
}

type AccessClaims struct {
	AccountType          string   `json:"account_type"`
	Status               string   `json:"status"`
	Roles                []string `json:"roles"`
	DepartmentID         string   `json:"department_id,omitempty"`
	Permissions          []string `json:"permissions"`
	AuthorizationVersion int64    `json:"authorization_version"`
	jwt.RegisteredClaims
}

func NewTokenManager(config TokenConfig) (*TokenManager, error) {
	if strings.TrimSpace(config.Issuer) == "" || strings.TrimSpace(config.Audience) == "" {
		return nil, fmt.Errorf("%w: issuer and audience are required", ErrInvalidTokenConfig)
	}
	if len(config.VerificationKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("%w: Ed25519 verification key is required", ErrInvalidTokenConfig)
	}
	if len(config.SigningKey) != 0 && len(config.SigningKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("%w: invalid Ed25519 signing key", ErrInvalidTokenConfig)
	}
	if len(config.SigningKey) != 0 && !bytes.Equal(config.SigningKey.Public().(ed25519.PublicKey), config.VerificationKey) {
		return nil, fmt.Errorf("%w: signing and verification keys do not match", ErrInvalidTokenConfig)
	}
	if config.TTL <= 0 {
		return nil, fmt.Errorf("%w: ttl must be positive", ErrInvalidTokenConfig)
	}

	return &TokenManager{
		issuer:          config.Issuer,
		audience:        config.Audience,
		signingKey:      append(ed25519.PrivateKey(nil), config.SigningKey...),
		verificationKey: append(ed25519.PublicKey(nil), config.VerificationKey...),
		ttl:             config.TTL,
	}, nil
}

func (m *TokenManager) Issue(principal Principal) (string, time.Time, error) {
	if principal.AccountID == "" || principal.Status != AccountStatusActive {
		return "", time.Time{}, ErrInactiveAccount
	}
	if len(m.signingKey) == 0 {
		return "", time.Time{}, ErrSigningUnavailable
	}

	now := time.Now().UTC()
	expiresAt := now.Add(m.ttl)
	claims := AccessClaims{
		AccountType:          principal.AccountType,
		Status:               principal.Status,
		Roles:                normalized(principal.Roles),
		DepartmentID:         principal.DepartmentID,
		Permissions:          normalized(principal.Permissions),
		AuthorizationVersion: principal.AuthorizationVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   principal.AccountID,
			Audience:  jwt.ClaimStrings{m.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	signed, err := token.SignedString(m.signingKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return signed, expiresAt, nil
}

func (m *TokenManager) Verify(raw string) (Principal, error) {
	claims := new(AccessClaims)
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodEdDSA {
			return nil, ErrInvalidToken
		}
		return m.verificationKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}))
	if err != nil || !token.Valid {
		return Principal{}, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if !claims.VerifyIssuer(m.issuer, true) || !claims.VerifyAudience(m.audience, true) {
		return Principal{}, ErrInvalidToken
	}
	if claims.Subject == "" || claims.Status != AccountStatusActive {
		return Principal{}, ErrInvalidToken
	}

	return Principal{
		AccountID:            claims.Subject,
		AccountType:          claims.AccountType,
		Status:               claims.Status,
		Roles:                normalized(claims.Roles),
		DepartmentID:         claims.DepartmentID,
		Permissions:          normalized(claims.Permissions),
		AuthorizationVersion: claims.AuthorizationVersion,
	}, nil
}

func normalized(values []string) []string {
	unique := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			unique[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(unique))
	for value := range unique {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
