package amap

import (
	"context"
	"net/url"

	"hospital/service/guidance/rpc/internal/routing"
)

// 驾车 v3 的首选方案与步行接口具有相同的 distance、duration、steps 结构。
func (c *Client) calculateDrivingRoute(ctx context.Context, origin, destination routing.LocationPoint) (routing.WalkingRoute, error) {
	query := commonRouteQuery(origin, destination)
	query.Set("strategy", "0")
	query.Set("extensions", "base")
	var payload walkingResponse
	if err := c.request(ctx, c.drivingEndpoint, query, &payload); err != nil {
		return routing.WalkingRoute{}, err
	}
	if err := payload.responseStatus.validate(); err != nil {
		return routing.WalkingRoute{}, err
	}
	if len(payload.Route.Paths) == 0 {
		return routing.WalkingRoute{}, routing.ErrNoRoute
	}
	return buildRoute(origin, destination, routing.RouteModeDriving, payload.Route.Paths[0])
}

func commonRouteQuery(origin, destination routing.LocationPoint) url.Values {
	query := url.Values{}
	query.Set("origin", coordinate(origin))
	query.Set("destination", coordinate(destination))
	if origin.ProviderPlaceID != "" {
		query.Set("origin_id", origin.ProviderPlaceID)
	}
	if destination.ProviderPlaceID != "" {
		query.Set("destination_id", destination.ProviderPlaceID)
	}
	return query
}
