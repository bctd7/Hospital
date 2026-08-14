package input

import "hospital/service/appointment/rpc/internal/manager/common"

// SaveReportTemplate 保存检查项目新报告的固定四字段默认正文。
type SaveReportTemplate struct {
	ItemID                  string
	Template                common.ReportContent
	ExpectedTemplateVersion int64
	Operation
}
