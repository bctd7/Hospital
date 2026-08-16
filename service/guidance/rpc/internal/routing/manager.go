package routing

import (
	"context"
	"math"
	"strings"
)

const (
	defaultPlaceLimit           = 10
	maximumPlaceLimit           = 20
	sameLocationThresholdMeters = 5
)

type PlaceSearchInput struct {
	Keyword string
	City    string
	Limit   int32
}

// Provider 定义路线业务需要的地图能力，当前由高德实现。
type Provider interface {
	SearchPlaces(context.Context, PlaceSearchInput) ([]LocationPoint, error)
	CalculateWalkingRoute(context.Context, LocationPoint, LocationPoint) (WalkingRoute, error)
}

// Manager 是地点搜索与步行路线计算的统一业务入口。
// 它负责参数规范化和通用业务校验，供应商包只处理外部 API 协议。
type Manager struct {
	provider Provider
}

func NewManager(provider Provider) (*Manager, error) {
	if provider == nil {
		return nil, ErrInvalid
	}
	return &Manager{provider: provider}, nil
}

func (m *Manager) SearchPlaces(ctx context.Context, input PlaceSearchInput) ([]LocationPoint, error) {
	input.Keyword = strings.TrimSpace(input.Keyword)
	input.City = strings.TrimSpace(input.City)
	if input.Keyword == "" || len([]rune(input.Keyword)) > 80 || len([]rune(input.City)) > 40 {
		return nil, ErrInvalid
	}
	if input.Limit == 0 {
		input.Limit = defaultPlaceLimit
	}
	if input.Limit < 1 || input.Limit > maximumPlaceLimit {
		return nil, ErrInvalid
	}
	return m.provider.SearchPlaces(ctx, input)
}

func (m *Manager) CalculateWalkingRoute(ctx context.Context, origin, destination LocationPoint) (WalkingRoute, error) {
	origin = normalizeLocation(origin)
	destination = normalizeLocation(destination)
	if !validLocation(origin) || !validLocation(destination) {
		return WalkingRoute{}, ErrInvalid
	}
	if distanceMeters(origin, destination) <= sameLocationThresholdMeters {
		return WalkingRoute{
			Origin: origin, Destination: destination,
			Polyline: []RoutePoint{
				{Latitude: origin.Latitude, Longitude: origin.Longitude},
				{Latitude: destination.Latitude, Longitude: destination.Longitude},
			},
			Provider: "local",
		}, nil
	}
	return m.provider.CalculateWalkingRoute(ctx, origin, destination)
}

func distanceMeters(origin, destination LocationPoint) float64 {
	const earthRadiusMeters = 6371000
	latitudeDelta := degreesToRadians(destination.Latitude - origin.Latitude)
	longitudeDelta := degreesToRadians(destination.Longitude - origin.Longitude)
	originLatitude := degreesToRadians(origin.Latitude)
	destinationLatitude := degreesToRadians(destination.Latitude)
	a := math.Sin(latitudeDelta/2)*math.Sin(latitudeDelta/2) +
		math.Cos(originLatitude)*math.Cos(destinationLatitude)*
			math.Sin(longitudeDelta/2)*math.Sin(longitudeDelta/2)
	return earthRadiusMeters * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func degreesToRadians(value float64) float64 {
	return value * math.Pi / 180
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
