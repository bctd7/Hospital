package guidanceplanning

import (
	"context"
	"strings"

	guidancev1 "hospital/contracts/gen/guidance/v1"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

type patientDisplay struct {
	name        string
	maskedPhone string
}

func currentPatientDisplay(ctx context.Context, svcCtx *svc.ServiceContext, requestID string) (patientDisplay, error) {
	value, err := svcCtx.Identity.GetAccountDisplayProfile(ctx, &identityv1.GetAccountDisplayProfileRequest{RequestId: requestID})
	if err != nil {
		return patientDisplay{}, err
	}
	name := strings.TrimSpace(value.GetNickname())
	if name == "" {
		name = "患者"
	}
	return patientDisplay{name: name, maskedPhone: strings.TrimSpace(value.GetMaskedPhone())}, nil
}

func smartPlansResponse(value *guidancev1.SmartAppointmentPlansResponse) *types.SmartAppointmentPlansResponse {
	response := &types.SmartAppointmentPlansResponse{Plans: make([]types.SmartAppointmentPlan, 0, len(value.GetPlans()))}
	for _, plan := range value.GetPlans() {
		items := make([]types.SmartAppointmentPlanItem, 0, len(plan.GetItems()))
		for _, item := range plan.GetItems() {
			items = append(items, smartPlanItemResponse(item))
		}
		response.Plans = append(response.Plans, types.SmartAppointmentPlan{
			PlanID: plan.GetPlanId(), Title: plan.GetTitle(), Summary: plan.GetSummary(), Items: items, ExpiresAt: plan.GetExpiresAt(),
		})
	}
	return response
}

func todayRecommendationResponse(value *guidancev1.TodayExaminationRecommendation) *types.TodayExaminationRecommendationResponse {
	response := &types.TodayExaminationRecommendationResponse{
		ServiceDate: value.GetServiceDate(), UpdatedAt: value.GetUpdatedAt(),
		Stages: make([]types.TodayRecommendationStage, 0, len(value.GetStages())),
	}
	for _, stage := range value.GetStages() {
		items := make([]types.SmartAppointmentPlanItem, 0, len(stage.GetItems()))
		for _, item := range stage.GetItems() {
			items = append(items, smartPlanItemResponse(item))
		}
		response.Stages = append(response.Stages, types.TodayRecommendationStage{
			StageNo: stage.GetStageNo(), Title: stage.GetTitle(), Status: stage.GetStatus(), Focus: stage.GetFocus(), Items: items,
		})
	}
	return response
}

func smartPlanItemResponse(item *guidancev1.SmartAppointmentPlanItem) types.SmartAppointmentPlanItem {
	return types.SmartAppointmentPlanItem{
		ItemID: item.GetItemId(), ItemName: item.GetItemName(), RoomID: item.GetRoomId(), RoomDisplayName: item.GetRoomDisplayName(),
		CampusID: item.GetCampusId(), Building: item.GetBuilding(), FloorNumber: item.GetFloorNumber(), RoomNumber: item.GetRoomNumber(),
		ServiceDate: item.GetServiceDate(), Session: item.GetSession(), EstimatedDurationMinutes: item.GetEstimatedDurationMinutes(), Reason: item.GetReason(),
	}
}
