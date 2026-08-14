package logic

import (
	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/manager/common"
)

func reportContent(in *appointmentv1.ExaminationReportContent) common.ReportContent {
	if in == nil {
		return common.ReportContent{}
	}
	return common.ReportContent{ObjectiveFindings: in.GetObjectiveFindings(), Impression: in.GetImpression(), Recommendation: in.GetRecommendation(), Notes: in.GetNotes()}
}

func reportVersionResponse(value common.ReportVersion) *appointmentv1.ExaminationReportVersion {
	response := &appointmentv1.ExaminationReportVersion{
		VersionId: value.VersionID, VersionNo: value.VersionNo, VersionKind: string(value.VersionKind), Status: string(value.Status),
		ObjectiveFindings: value.ObjectiveFindings, Impression: value.Impression,
		Recommendation: value.Recommendation, Notes: value.Notes, CorrectionReason: value.CorrectionReason,
		AuthoredBy: value.AuthoredBy, AuthoredByDisplayName: value.AuthoredByDisplayName,
		PublishedBy: value.PublishedBy, PublishedByDisplayName: value.PublishedByDisplayName,
		CreatedAt: value.CreatedAt.UTC().Format(timeLayout), UpdatedAt: value.UpdatedAt.UTC().Format(timeLayout),
	}
	if value.PublishedAt != nil {
		response.PublishedAt = value.PublishedAt.UTC().Format(timeLayout)
	}
	return response
}

func reportResponse(value common.ExaminationReport) *appointmentv1.ExaminationReport {
	response := &appointmentv1.ExaminationReport{
		ReportId: value.ReportID, BookingId: value.BookingID, PatientAccountId: value.PatientAccountID,
		PatientDisplayName: value.PatientDisplayName, PatientPhoneMasked: value.PatientPhoneMasked,
		DepartmentId: value.DepartmentID, DepartmentName: value.DepartmentName,
		ItemId: value.ItemID, ItemName: value.ItemName,
		RoomId: value.RoomID, CampusId: value.CampusID, CampusName: value.CampusName, Building: value.Building,
		FloorNumber: value.FloorNumber, RoomNumber: value.RoomNumber, RoomDisplayName: value.RoomDisplayName,
		Status: string(value.Status), PerformedBy: value.PerformedBy,
		PerformedByDisplayName: value.PerformedByDisplayName, Version: value.Version,
		CreatedAt: value.CreatedAt.UTC().Format(timeLayout), UpdatedAt: value.UpdatedAt.UTC().Format(timeLayout),
	}
	if value.ExaminationStartedAt != nil {
		response.ExaminationStartedAt = value.ExaminationStartedAt.UTC().Format(timeLayout)
	}
	if value.ExaminationCompletedAt != nil {
		response.ExaminationCompletedAt = value.ExaminationCompletedAt.UTC().Format(timeLayout)
	}
	if value.CurrentVersion != nil {
		response.CurrentVersion = reportVersionResponse(*value.CurrentVersion)
	}
	return response
}
