package logic

import (
	"context"
	"errors"

	v1_guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/guidance/rpc/internal/routing"
	"hospital/service/guidance/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CalculateWalkingRouteLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCalculateWalkingRouteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CalculateWalkingRouteLogic {
	return &CalculateWalkingRouteLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CalculateWalkingRouteLogic) CalculateWalkingRoute(in *v1_guidancev1.CalculateWalkingRouteRequest) (*v1_guidancev1.WalkingRoute, error) {
	if _, err := guidancePrincipal(l.ctx); err != nil {
		return nil, err
	}
	if in == nil || in.Origin == nil || in.Destination == nil {
		return nil, mapRPCError(routing.ErrInvalid)
	}
	origin := routeLocation(in.Origin)
	destination := routeLocation(in.Destination)
	result, err := l.svcCtx.RoutingManager.CalculateRoute(l.ctx, routing.RouteMode(in.Mode), origin, destination)
	if err != nil {
		return nil, mapRPCError(err)
	}
	return walkingRouteResponse(result), nil
}

func routeLocation(point *v1_guidancev1.LocationPoint) routing.LocationPoint {
	if point == nil {
		return routing.LocationPoint{}
	}
	return routing.LocationPoint{
		Name: point.Name, Address: point.Address, Latitude: point.Latitude,
		Longitude: point.Longitude, ProviderPlaceID: point.ProviderPlaceId,
	}
}

func walkingRouteResponse(route routing.WalkingRoute) *v1_guidancev1.WalkingRoute {
	response := &v1_guidancev1.WalkingRoute{
		Origin: routeLocationResponse(route.Origin), Destination: routeLocationResponse(route.Destination),
		DistanceMeters: route.DistanceMeters, DurationSeconds: route.DurationSeconds, Provider: route.Provider,
		Mode:     string(route.Mode),
		Polyline: make([]*v1_guidancev1.RoutePoint, 0, len(route.Polyline)),
		Steps:    make([]*v1_guidancev1.WalkingRouteStep, 0, len(route.Steps)),
	}
	for _, point := range route.Polyline {
		response.Polyline = append(response.Polyline, &v1_guidancev1.RoutePoint{Latitude: point.Latitude, Longitude: point.Longitude})
	}
	for _, step := range route.Steps {
		response.Steps = append(response.Steps, &v1_guidancev1.WalkingRouteStep{
			Instruction: step.Instruction, RoadName: step.RoadName,
			DistanceMeters: step.DistanceMeters, DurationSeconds: step.DurationSeconds,
		})
	}
	return response
}

func routeLocationResponse(point routing.LocationPoint) *v1_guidancev1.LocationPoint {
	return &v1_guidancev1.LocationPoint{
		Name: point.Name, Address: point.Address, Latitude: point.Latitude,
		Longitude: point.Longitude, ProviderPlaceId: point.ProviderPlaceID,
	}
}

func mapRPCError(err error) error {
	switch {
	case errors.Is(err, routing.ErrInvalid):
		return status.Error(codes.InvalidArgument, "invalid map request")
	case errors.Is(err, routing.ErrNoRoute):
		return status.Error(codes.NotFound, "route not found")
	case errors.Is(err, routing.ErrUnavailable), errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return status.Error(codes.Unavailable, "map service unavailable")
	default:
		return status.Error(codes.Internal, "map provider failed")
	}
}
