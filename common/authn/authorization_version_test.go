package authn

import (
	"context"
	"errors"
	"testing"
)

func TestAuthorizationVersionValidator(t *testing.T) {
	tests := []struct {
		name      string
		principal Principal
		reader    fakeAuthorizationVersionReader
		want      error
	}{
		{
			name:      "matching version",
			principal: Principal{AccountID: "account-1", AuthorizationVersion: 3},
			reader:    fakeAuthorizationVersionReader{version: 3},
		},
		{
			name:      "stale version",
			principal: Principal{AccountID: "account-1", AuthorizationVersion: 2},
			reader:    fakeAuthorizationVersionReader{version: 3},
			want:      ErrAuthorizationVersionStale,
		},
		{
			name:      "reader failure",
			principal: Principal{AccountID: "account-1", AuthorizationVersion: 3},
			reader:    fakeAuthorizationVersionReader{err: errors.New("redis unavailable")},
			want:      ErrAuthorizationVersionUnavailable,
		},
		{
			name:      "missing projection",
			principal: Principal{AccountID: "account-1", AuthorizationVersion: 3},
			reader:    fakeAuthorizationVersionReader{},
			want:      ErrAuthorizationVersionUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := NewAuthorizationVersionValidator(tt.reader)
			if err != nil {
				t.Fatal(err)
			}
			err = validator.ValidatePrincipal(context.Background(), tt.principal)
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

type fakeAuthorizationVersionReader struct {
	version int64
	err     error
}

func (r fakeAuthorizationVersionReader) CurrentAuthorizationVersion(context.Context, string) (int64, error) {
	return r.version, r.err
}
