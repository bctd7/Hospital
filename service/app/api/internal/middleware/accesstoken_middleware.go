// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package middleware

import (
	"net/http"

	"hospital/common/authn"
)

type AccessTokenMiddleware struct {
	handle func(http.HandlerFunc) http.HandlerFunc
}

func NewAccessTokenMiddleware(manager *authn.TokenManager) *AccessTokenMiddleware {
	return &AccessTokenMiddleware{handle: authn.HTTPMiddleware(manager)}
}

func (m *AccessTokenMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return m.handle(next)
}
