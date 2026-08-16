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

type SearchGuidancePlacesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSearchGuidancePlacesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchGuidancePlacesLogic {
	return &SearchGuidancePlacesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SearchGuidancePlacesLogic) SearchGuidancePlaces(req *types.SearchGuidancePlacesRequest) (resp *types.SearchGuidancePlacesResponse, err error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	result, err := l.svcCtx.Guidance.SearchPlaces(ctx, &guidancev1.SearchPlacesRequest{
		Keyword: req.Keyword, City: req.City, Limit: req.Limit,
		RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	response := &types.SearchGuidancePlacesResponse{Places: make([]types.GuidanceLocationPoint, 0, len(result.GetPlaces()))}
	for _, place := range result.GetPlaces() {
		response.Places = append(response.Places, routeLocationResponse(place))
	}
	return response, nil
}
