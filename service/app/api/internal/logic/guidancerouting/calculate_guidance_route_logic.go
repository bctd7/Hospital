// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package guidancerouting

import (
	"context"

	"hospital/common/observability/logging"
	guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CalculateGuidanceRouteLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCalculateGuidanceRouteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CalculateGuidanceRouteLogic {
	return &CalculateGuidanceRouteLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CalculateGuidanceRouteLogic) CalculateGuidanceRoute(req *types.CalculateGuidanceRouteRequest) (resp *types.GuidanceRouteResponse, err error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	result, err := l.svcCtx.Guidance.CalculateWalkingRoute(ctx, &guidancev1.CalculateWalkingRouteRequest{
		Origin: routeLocationRequest(req.Origin), Destination: routeLocationRequest(req.Destination),
		Mode: req.Mode, RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return routeResponse(result), nil
}

func routeLocationRequest(point types.GuidanceLocationPoint) *guidancev1.LocationPoint {
	return &guidancev1.LocationPoint{Name: point.Name, Address: point.Address, Latitude: point.Latitude, Longitude: point.Longitude, ProviderPlaceId: point.ProviderPlaceID}
}

func routeResponse(route *guidancev1.WalkingRoute) *types.GuidanceRouteResponse {
	if route == nil {
		return &types.GuidanceRouteResponse{}
	}
	response := &types.GuidanceRouteResponse{
		Origin: routeLocationResponse(route.Origin), Destination: routeLocationResponse(route.Destination),
		DistanceMeters: route.DistanceMeters, DurationSeconds: route.DurationSeconds, Provider: route.Provider, Mode: route.Mode,
		Polyline: make([]types.GuidanceRoutePoint, 0, len(route.Polyline)), Steps: make([]types.GuidanceRouteStepResponse, 0, len(route.Steps)),
	}
	for _, point := range route.Polyline {
		if point != nil {
			response.Polyline = append(response.Polyline, types.GuidanceRoutePoint{Latitude: point.Latitude, Longitude: point.Longitude})
		}
	}
	for _, step := range route.Steps {
		if step != nil {
			response.Steps = append(response.Steps, types.GuidanceRouteStepResponse{Instruction: step.Instruction, RoadName: step.RoadName, DistanceMeters: step.DistanceMeters, DurationSeconds: step.DurationSeconds})
		}
	}
	return response
}

func routeLocationResponse(point *guidancev1.LocationPoint) types.GuidanceLocationPoint {
	if point == nil {
		return types.GuidanceLocationPoint{}
	}
	return types.GuidanceLocationPoint{Name: point.Name, Address: point.Address, Latitude: point.Latitude, Longitude: point.Longitude, ProviderPlaceID: point.ProviderPlaceId}
}
