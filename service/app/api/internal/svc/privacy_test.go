package svc

import (
	"slices"
	"testing"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	identityv1 "hospital/contracts/gen/identity/v1"
)

func TestSensitiveIdentityRPCMethodsIncludeAdminPhoneSearch(t *testing.T) {
	method := identityv1.IdentityService_SearchAdminAccountByPhone_FullMethodName
	if !slices.Contains(identityRPCMethodsWithSensitiveContent(), method) {
		t.Fatalf("sensitive RPC method %q is missing from app-api client log suppression", method)
	}
}

func TestSensitiveAppointmentRPCMethodsIncludeReportReadsAndWrites(t *testing.T) {
	methods := appointmentRPCMethodsWithSensitiveContent()
	for _, method := range []string{
		appointmentv1.AppointmentService_GetExaminationItemReportTemplate_FullMethodName,
		appointmentv1.AppointmentService_SaveExaminationItemReportTemplate_FullMethodName,
		appointmentv1.AppointmentService_SaveExaminationReportDraft_FullMethodName,
		appointmentv1.AppointmentService_CorrectExaminationReport_FullMethodName,
		appointmentv1.AppointmentService_ListExaminationReports_FullMethodName,
		appointmentv1.AppointmentService_GetMyExaminationReport_FullMethodName,
		appointmentv1.AppointmentService_ListMyExaminationReports_FullMethodName,
	} {
		if !slices.Contains(methods, method) {
			t.Fatalf("sensitive report RPC method %q is missing from client log suppression", method)
		}
	}
}
