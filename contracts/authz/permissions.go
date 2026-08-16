package authz

const (
	PermissionIdentityAuthorizationManage = "identity.authorization.manage"
	PermissionIdentityAccountManage       = "identity.account.manage"
	PermissionIdentityDepartmentManage    = "identity.department.manage"

	PermissionAppointmentRead       = "appointment.read"
	PermissionAppointmentCreate     = "appointment.create"
	PermissionAppointmentUpdate     = "appointment.update"
	PermissionAppointmentCancel     = "appointment.cancel"
	PermissionAppointmentReschedule = "appointment.reschedule"

	PermissionPlanningRead   = "planning.read"
	PermissionPlanningCreate = "planning.create"
	PermissionPlanningAdjust = "planning.adjust"

	PermissionNavigationRead    = "navigation.read"
	PermissionNavigationEdit    = "navigation.edit"
	PermissionNavigationPublish = "navigation.publish"

	PermissionReportRead     = "report.read"
	PermissionReportPublish  = "report.publish"
	PermissionReportCorrect  = "report.correct"
	PermissionReportDownload = "report.download"
	PermissionReportExport   = "report.export"

	PermissionRuleRead    = "rule.read"
	PermissionRuleEdit    = "rule.edit"
	PermissionRuleReview  = "rule.review"
	PermissionRulePublish = "rule.publish"
)

type Permission struct {
	Code    string
	Service string
}

var AllPermissions = []Permission{
	{Code: PermissionIdentityAuthorizationManage, Service: "identity"},
	{Code: PermissionIdentityAccountManage, Service: "identity"},
	{Code: PermissionIdentityDepartmentManage, Service: "identity"},
	{Code: PermissionAppointmentRead, Service: "appointment"},
	{Code: PermissionAppointmentCreate, Service: "appointment"},
	{Code: PermissionAppointmentUpdate, Service: "appointment"},
	{Code: PermissionAppointmentCancel, Service: "appointment"},
	{Code: PermissionAppointmentReschedule, Service: "appointment"},
	{Code: PermissionPlanningRead, Service: "planning"},
	{Code: PermissionPlanningCreate, Service: "planning"},
	{Code: PermissionPlanningAdjust, Service: "planning"},
	{Code: PermissionNavigationRead, Service: "navigation"},
	{Code: PermissionNavigationEdit, Service: "navigation"},
	{Code: PermissionNavigationPublish, Service: "navigation"},
	{Code: PermissionReportRead, Service: "report"},
	{Code: PermissionReportPublish, Service: "report"},
	{Code: PermissionReportCorrect, Service: "report"},
	{Code: PermissionReportDownload, Service: "report"},
	{Code: PermissionReportExport, Service: "report"},
	{Code: PermissionRuleRead, Service: "guidance"},
	{Code: PermissionRuleEdit, Service: "guidance"},
	{Code: PermissionRuleReview, Service: "guidance"},
	{Code: PermissionRulePublish, Service: "guidance"},
}
