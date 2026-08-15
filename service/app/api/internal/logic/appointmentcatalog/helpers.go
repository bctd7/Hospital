package appointmentcatalog

import (
	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/app/api/internal/types"
)

func examinationItemResponse(item *appointmentv1.ExaminationItem) *types.ExaminationItemResponse {
	if item == nil {
		return nil
	}
	return &types.ExaminationItemResponse{
		ItemID:                   item.GetItemId(),
		OwnerDepartmentID:        item.GetOwnerDepartmentId(),
		Name:                     item.GetName(),
		Description:              item.GetDescription(),
		EstimatedDurationMinutes: item.GetEstimatedDurationMinutes(),
		Status:                   item.GetStatus(),
		Version:                  item.GetVersion(),
		CreatedAt:                item.GetCreatedAt(),
		UpdatedAt:                item.GetUpdatedAt(),
	}
}

func examinationItemsResponse(value *appointmentv1.ListExaminationItemsResponse) *types.ListExaminationItemsResponse {
	items := make([]types.ExaminationItemResponse, 0, len(value.GetItems()))
	for _, item := range value.GetItems() {
		response := examinationItemResponse(item)
		if response != nil {
			items = append(items, *response)
		}
	}
	return &types.ListExaminationItemsResponse{
		Items: items, Page: value.GetPage(), PageSize: value.GetPageSize(), Total: value.GetTotal(),
	}
}

func changeStatusRequest(req *types.ChangeExaminationItemStatusRequest, requestID string) *appointmentv1.ChangeExaminationItemStatusRequest {
	return &appointmentv1.ChangeExaminationItemStatusRequest{
		ItemId: req.ItemID, ExpectedVersion: req.ExpectedVersion,
		OperationId: req.OperationID, RequestId: requestID,
	}
}
