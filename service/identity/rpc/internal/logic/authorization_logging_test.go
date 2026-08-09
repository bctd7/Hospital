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

func TestAuthorizationFailureUsesFunctionalLogging(t *testing.T) {
	var logs bytes.Buffer
	restoreIdentityLogWriter(t, &logs)

	store := deniedAuthorizationStore{}
	ctx := authn.ContextWithPrincipal(context.Background(), authn.Principal{
		AccountID: "20000000-0000-0000-0000-000000000001",
		Status:    authn.AccountStatusActive,
	})
	logic := NewAssignRoleLogic(ctx, &svc.ServiceContext{
		AuthorizationManager: authorization.NewManager(store),
	})
	_, err := logic.AssignRole(&identityv1.AssignRoleRequest{
		TargetAccountId: "20000000-0000-0000-0000-000000000002",
		RoleCode:        authn.RoleDepartmentDoctor,
		OperationId:     "20000000-0000-0000-0000-000000000003",
		RequestId:       "authorization-request-1",
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", err)
	}

	output := logs.String()
	for _, expected := range []string{
		`"level":"error"`,
		`"event":"identity.role.assigned"`,
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

type deniedAuthorizationStore struct{}

func (deniedAuthorizationStore) GetAuthorizationContext(_ context.Context, accountID string) (authn.Principal, error) {
	return authn.Principal{AccountID: accountID, AccountType: authn.AccountTypeStaff, Status: authn.AccountStatusActive}, nil
}

func (s deniedAuthorizationStore) WithinTransaction(ctx context.Context, fn func(authorization.TxStore) error) error {
	return fn(s)
}

func (deniedAuthorizationStore) FindOperation(context.Context, string) (authorization.Operation, bool, error) {
	return authorization.Operation{}, false, nil
}

func (deniedAuthorizationStore) SetRole(context.Context, string, string) error {
	panic("SetRole must not be called for a denied operator")
}

func (deniedAuthorizationStore) SetDepartment(context.Context, string, string) error {
	panic("SetDepartment must not be called for a denied operator")
}

func (deniedAuthorizationStore) SetAccountStatus(context.Context, string, string) error {
	panic("SetAccountStatus must not be called for a denied operator")
}

func (deniedAuthorizationStore) RecordChange(context.Context, authorization.Change) error {
	panic("RecordChange must not be called for a denied operator")
}
