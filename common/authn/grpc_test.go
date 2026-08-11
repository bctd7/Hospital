package authn

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestUnaryServerInterceptor(t *testing.T) {
	manager := newTestTokenManager(t)
	raw, _, err := manager.Issue(Principal{
		AccountID: "account-1", Status: AccountStatusActive, AuthorizationVersion: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+raw))
	_, err = UnaryServerInterceptor(manager, acceptingPrincipalValidator{})(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/identity/Get"}, func(ctx context.Context, _ any) (any, error) {
		principal, err := PrincipalFromContext(ctx)
		if err != nil {
			return nil, err
		}
		if principal.AccountID != "account-1" {
			t.Fatalf("unexpected principal: %#v", principal)
		}
		return "ok", nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestUnaryServerInterceptorRejectsMissingToken(t *testing.T) {
	manager := newTestTokenManager(t)
	_, err := UnaryServerInterceptor(manager, acceptingPrincipalValidator{})(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/identity/Get"}, func(context.Context, any) (any, error) {
		t.Fatal("handler must not run")
		return nil, nil
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", err)
	}
}

func TestUnaryServerInterceptorRejectsInvalidAuthorizationVersion(t *testing.T) {
	manager := newTestTokenManager(t)
	raw, _, err := manager.Issue(Principal{
		AccountID: "account-1", Status: AccountStatusActive, AuthorizationVersion: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+raw))
	_, err = UnaryServerInterceptor(manager, rejectingPrincipalValidator{})(
		ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/identity/Get"},
		func(context.Context, any) (any, error) {
			t.Fatal("handler must not run")
			return nil, nil
		},
	)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", err)
	}
}

func TestUnaryServerInterceptorAllowsExplicitPublicMethod(t *testing.T) {
	manager := newTestTokenManager(t)
	const method = "/identity/RefreshAccessToken"
	result, err := UnaryServerInterceptor(manager, acceptingPrincipalValidator{}, method)(
		context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: method},
		func(context.Context, any) (any, error) { return "ok", nil },
	)
	if err != nil || result != "ok" {
		t.Fatalf("public method was not allowed: result=%v err=%v", result, err)
	}
}
