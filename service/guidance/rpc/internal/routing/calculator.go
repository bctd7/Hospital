package routing

import (
	"context"
	"math"
	"strings"
)

type Provider interface {
	CalculateWalkingRoute(context.Context, LocationPoint, LocationPoint) (WalkingRoute, error)
}

// Calculator 只负责校验统一地点契约并调用道路路线供应商，不参与检查项目排序。
type Calculator struct {
	provider Provider
}

func NewCalculator(provider Provider) (*Calculator, error) {
	if provider == nil {
		return nil, ErrInvalid
	}
	return &Calculator{provider: provider}, nil
}

func (c *Calculator) CalculateWalkingRoute(ctx context.Context, origin, destination LocationPoint) (WalkingRoute, error) {
	origin = normalizeLocation(origin)
	destination = normalizeLocation(destination)
	if !validLocation(origin) || !validLocation(destination) {
		return WalkingRoute{}, ErrInvalid
	}
	return c.provider.CalculateWalkingRoute(ctx, origin, destination)
}

func normalizeLocation(point LocationPoint) LocationPoint {
	point.Name = strings.TrimSpace(point.Name)
	point.Address = strings.TrimSpace(point.Address)
	point.ProviderPlaceID = strings.TrimSpace(point.ProviderPlaceID)
	return point
}

func validLocation(point LocationPoint) bool {
	return point.Name != "" &&
		!math.IsNaN(point.Latitude) && !math.IsInf(point.Latitude, 0) &&
		!math.IsNaN(point.Longitude) && !math.IsInf(point.Longitude, 0) &&
		point.Latitude >= -90 && point.Latitude <= 90 &&
		point.Longitude >= -180 && point.Longitude <= 180
}
