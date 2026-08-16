package logic

import (
	"context"

	guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/guidance/rpc/internal/routing"
	"hospital/service/guidance/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchPlacesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSearchPlacesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchPlacesLogic {
	return &SearchPlacesLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *SearchPlacesLogic) SearchPlaces(in *guidancev1.SearchPlacesRequest) (*guidancev1.SearchPlacesResponse, error) {
	if _, err := guidancePrincipal(l.ctx); err != nil {
		return nil, err
	}
	if in == nil {
		return nil, mapRPCError(routing.ErrInvalid)
	}
	places, err := l.svcCtx.PlaceFinder.Search(l.ctx, routing.PlaceSearchInput{
		Keyword: in.Keyword, City: in.City, Limit: in.Limit,
	})
	if err != nil {
		return nil, mapRPCError(err)
	}
	response := &guidancev1.SearchPlacesResponse{Places: make([]*guidancev1.LocationPoint, 0, len(places))}
	for _, place := range places {
		response.Places = append(response.Places, routeLocationResponse(place))
	}
	return response, nil
}
