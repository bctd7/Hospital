package input

import "hospital/service/appointment/rpc/internal/manager/common"

type StartExamination struct {
	BookingID        string
	ExpectedVersion  int64
	ActorDisplayName string
	Operation
}

type EndExamination struct {
	BookingID        string
	ExpectedVersion  int64
	ActorDisplayName string
	Operation
}

type SaveReportDraft struct {
	BookingID             string
	Content               common.ReportContent
	ExpectedReportVersion int64
	ActorDisplayName      string
	Operation
}

type CompleteAndPublishReport struct {
	BookingID              string
	Content                common.ReportContent
	ExpectedBookingVersion int64
	ExpectedReportVersion  int64
	ActorDisplayName       string
	DepartmentName         string
	CampusName             string
	Operation
}

type CorrectReport struct {
	ReportID              string
	Content               common.ReportContent
	CorrectionReason      string
	ExpectedReportVersion int64
	ActorDisplayName      string
	Operation
}
