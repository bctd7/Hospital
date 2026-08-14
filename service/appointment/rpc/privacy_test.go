package main

import (
	"slices"
	"testing"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
)

func TestReportRPCMethodsSuppressSensitiveContentLogging(t *testing.T) {
	methods := reportRPCMethodsWithSensitiveContent()
	for _, method := range []string{
		appointmentv1.AppointmentService_GetExaminationItemReportTemplate_FullMethodName,
		appointmentv1.AppointmentService_SaveExaminationItemReportTemplate_FullMethodName,
		appointmentv1.AppointmentService_SaveExaminationReportDraft_FullMethodName,
		appointmentv1.AppointmentService_CompleteAndPublishExaminationReport_FullMethodName,
		appointmentv1.AppointmentService_CorrectExaminationReport_FullMethodName,
		appointmentv1.AppointmentService_ListExaminationReports_FullMethodName,
		appointmentv1.AppointmentService_GetMyExaminationReport_FullMethodName,
	} {
		if !slices.Contains(methods, method) {
			t.Fatalf("sensitive report RPC method %q is missing from server log suppression", method)
		}
	}
}
