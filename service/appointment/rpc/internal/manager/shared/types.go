package shared

import "hospital/service/appointment/rpc/internal/manager/common"

type ListProjectsQuery struct {
	OwnerDepartmentID string
	Status            common.Status
	Page              int64
	PageSize          int64
}

type ListProjectsResult struct {
	Items    []common.ExaminationItem
	Page     int64
	PageSize int64
	Total    int64
}
