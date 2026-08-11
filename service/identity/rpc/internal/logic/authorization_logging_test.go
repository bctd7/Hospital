package logic

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"hospital/common/authn"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/authorization"
	"hospital/service/identity/rpc/internal/svc"
)

func TestAuthorizationContextReadFailureUsesFunctionalLogging(t *testing.T) {
	var logs bytes.Buffer
	restoreIdentityLogWriter(t, &logs)

	manager, err := authorization.NewManager(deniedAuthorizationReadStore{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := authn.ContextWithPrincipal(context.Background(), authn.Principal{
		AccountID: "20000000-0000-0000-0000-000000000001",
		Status:    authn.AccountStatusActive,
	})
	logic := NewGetAuthorizationContextLogic(ctx, &svc.ServiceContext{AuthorizationManager: manager})
	_, err = logic.GetAuthorizationContext(&identityv1.GetAuthorizationContextRequest{
		AccountId: "20000000-0000-0000-0000-000000000002",
		RequestId: "authorization-request-1",
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", err)
	}

	output := logs.String()
	for _, expected := range []string{
		`"level":"error"`,
		`"event":"identity.authorization.context.read"`,
		`"request_id":"authorization-request-1"`,
		`"operator_account_id":"20000000-0000-0000-0000-000000000001"`,
		`"target_account_id":"20000000-0000-0000-0000-000000000002"`,
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("authorization log does not contain %q: %s", expected, output)
		}
	}
}

func restoreIdentityLogWriter(t *testing.T, destination io.Writer) {
	t.Helper()
	previousWriter := logx.Reset()
	logx.SetWriter(logx.NewWriter(destination))
	t.Cleanup(func() {
		logx.Reset()
		if previousWriter != nil {
			logx.SetWriter(previousWriter)
		}
	})
}

type deniedAuthorizationReadStore struct{}

func (deniedAuthorizationReadStore) GetAuthorizationContext(context.Context, string) (authn.Principal, error) {
	panic("store must not be called when the operator lacks read permission")
}
