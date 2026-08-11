package organizationadmin

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"hospital/common/authn"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
	"hospital/service/identity/rpc/identityservice"
)

func TestListOrganizationUnitsLogicForwardsAccessToken(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tokenManager, err := authn.NewTokenManager(authn.TokenConfig{
		Issuer: "test", Audience: "test", SigningKey: privateKey,
		VerificationKey: publicKey, TTL: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, _, err := tokenManager.Issue(authn.Principal{
		AccountID: "50000000-0000-0000-0000-000000000001",
		Status:    authn.AccountStatusActive, AuthorizationVersion: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	client := &adminIdentityClient{expectedToken: raw}
	serviceContext := &svc.ServiceContext{Identity: client}
	var response *types.ListOrganizationUnitsResponse
	var logicErr error
	handler := authn.HTTPMiddleware(tokenManager, acceptingValidator{})(func(_ http.ResponseWriter, request *http.Request) {
		response, logicErr = NewListOrganizationUnitsLogic(request.Context(), serviceContext).ListOrganizationUnits(
			&types.ListOrganizationUnitsRequest{UnitType: "campus", ParentID: "hospital-1"},
		)
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+raw)
	handler(httptest.NewRecorder(), request)

	if logicErr != nil {
		t.Fatal(logicErr)
	}
	if len(response.Items) != 1 || response.Items[0].UnitID != "campus-1" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

type acceptingValidator struct{}

func (acceptingValidator) ValidatePrincipal(context.Context, authn.Principal) error { return nil }

type adminIdentityClient struct {
	identityservice.IdentityService
	expectedToken string
}

func (c *adminIdentityClient) ListOrganizationUnits(ctx context.Context, _ *identityv1.ListOrganizationUnitsRequest, _ ...grpc.CallOption) (*identityv1.ListOrganizationUnitsResponse, error) {
	values, ok := metadata.FromOutgoingContext(ctx)
	if !ok || len(values.Get("authorization")) != 1 || values.Get("authorization")[0] != "Bearer "+c.expectedToken {
		return nil, statusError("missing forwarded authorization metadata")
	}
	return &identityv1.ListOrganizationUnitsResponse{Items: []*identityv1.AdminOrganizationUnit{{
		UnitId: "campus-1", ParentId: "hospital-1", UnitType: "campus",
		Code: "CAMPUS-A", Name: "Main Campus", Status: "active", Version: 1,
	}}}, nil
}

type testError string

func (e testError) Error() string { return string(e) }

func statusError(message string) error { return testError(message) }
