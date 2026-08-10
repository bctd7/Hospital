package organizationdirectory

import (
	"context"
	"testing"

	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
	"hospital/service/identity/rpc/identityservice"

	"google.golang.org/grpc"
)

func TestPublicOrganizationDirectoryLogic(t *testing.T) {
	client := &directoryIdentityClient{}
	serviceContext := &svc.ServiceContext{Identity: client}

	contextResponse, err := NewGetOrganizationContextLogic(context.Background(), serviceContext).GetOrganizationContext()
	if err != nil {
		t.Fatal(err)
	}
	if contextResponse.Hospital.HospitalID != "hospital-1" || len(contextResponse.Campuses) != 1 {
		t.Fatalf("unexpected context response: %#v", contextResponse)
	}

	departments, err := NewListDepartmentsLogic(context.Background(), serviceContext).ListDepartments(
		&types.ListDepartmentsRequest{CampusID: "campus-1"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(departments.Items) != 1 || departments.Items[0].CampusName != "Main Campus" {
		t.Fatalf("unexpected departments: %#v", departments)
	}
}

type directoryIdentityClient struct {
	identityservice.IdentityService
}

func (*directoryIdentityClient) GetOrganizationContext(context.Context, *identityv1.GetOrganizationContextRequest, ...grpc.CallOption) (*identityv1.OrganizationContext, error) {
	return &identityv1.OrganizationContext{
		Hospital: &identityv1.HospitalSummary{HospitalId: "hospital-1", Code: "HOSPITAL", Name: "Hospital", Version: 1},
		Campuses: []*identityv1.CampusSummary{{
			CampusId: "campus-1", HospitalId: "hospital-1", Code: "CAMPUS-A", Name: "Main Campus",
			DepartmentCount: 1, Status: "active", Version: 1,
		}},
	}, nil
}

func (*directoryIdentityClient) ListDepartments(_ context.Context, request *identityv1.ListDepartmentsRequest, _ ...grpc.CallOption) (*identityv1.ListDepartmentsResponse, error) {
	return &identityv1.ListDepartmentsResponse{Items: []*identityv1.DepartmentSummary{{
		DepartmentId: "department-1", CampusId: request.GetCampusId(), CampusName: "Main Campus",
		Code: "DEPT-A", Name: "Cardiology", DoctorCount: 2, Status: "active", Version: 1,
	}}}, nil
}
