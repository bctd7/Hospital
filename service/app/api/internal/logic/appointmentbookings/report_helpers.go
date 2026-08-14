package appointmentbookings

import (
	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/app/api/internal/types"
)

func reportContent(objectiveFindings, impression, recommendation, notes string) *appointmentv1.ExaminationReportContent {
	return &appointmentv1.ExaminationReportContent{ObjectiveFindings: objectiveFindings, Impression: impression, Recommendation: recommendation, Notes: notes}
}

func reportVersion(value *appointmentv1.ExaminationReportVersion) types.ReportVersionResponse {
	if value == nil {
		return types.ReportVersionResponse{}
	}
	return types.ReportVersionResponse{
		VersionID: value.VersionId, VersionNo: value.VersionNo, VersionKind: value.VersionKind, Status: value.Status,
		ObjectiveFindings: value.ObjectiveFindings, Impression: value.Impression,
		Recommendation: value.Recommendation, Notes: value.Notes, CorrectionReason: value.CorrectionReason,
		AuthoredBy: value.AuthoredBy, AuthoredByDisplayName: value.AuthoredByDisplayName,
		PublishedBy: value.PublishedBy, PublishedByDisplayName: value.PublishedByDisplayName,
		PublishedAt: value.PublishedAt,
		CreatedAt:   value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}

func examinationReport(value *appointmentv1.ExaminationReport) *types.ExaminationReportResponse {
	if value == nil {
		return nil
	}
	return &types.ExaminationReportResponse{
		ReportID: value.ReportId, BookingID: value.BookingId, PatientAccountID: value.PatientAccountId,
		PatientDisplayName: value.PatientDisplayName, PatientPhoneMasked: value.PatientPhoneMasked,
		DepartmentID: value.DepartmentId, DepartmentName: value.DepartmentName,
		ItemID: value.ItemId, ItemName: value.ItemName,
		RoomID: value.RoomId, CampusID: value.CampusId, CampusName: value.CampusName, Building: value.Building,
		FloorNumber: value.FloorNumber, RoomNumber: value.RoomNumber, RoomDisplayName: value.RoomDisplayName,
		Status: value.Status, PerformedBy: value.PerformedBy, PerformedByDisplayName: value.PerformedByDisplayName,
		ExaminationStartedAt: value.ExaminationStartedAt, ExaminationCompletedAt: value.ExaminationCompletedAt,
		Version: value.Version, CurrentVersion: reportVersion(value.CurrentVersion),
		CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}
