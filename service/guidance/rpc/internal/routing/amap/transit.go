package amap

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"hospital/service/guidance/rpc/internal/routing"
)

// 公交路线只用于院外首段。当前产品固定在上海，city 使用上海城市编码。
func (c *Client) calculateTransitRoute(ctx context.Context, origin, destination routing.LocationPoint) (routing.WalkingRoute, error) {
	query := url.Values{}
	query.Set("origin", coordinate(origin))
	query.Set("destination", coordinate(destination))
	query.Set("city", "021")
	query.Set("cityd", "021")
	query.Set("strategy", "0")
	query.Set("extensions", "base")
	var payload transitResponse
	if err := c.request(ctx, c.transitEndpoint, query, &payload); err != nil {
		return routing.WalkingRoute{}, err
	}
	if err := payload.responseStatus.validate(); err != nil {
		return routing.WalkingRoute{}, err
	}
	if len(payload.Route.Transits) == 0 {
		return routing.WalkingRoute{}, routing.ErrNoRoute
	}
	return buildTransitRoute(origin, destination, payload.Route.Transits[0])
}

type transitResponse struct {
	responseStatus
	Route struct {
		Transits []transitPlan `json:"transits"`
	} `json:"route"`
}

type transitPlan struct {
	Duration        string           `json:"duration"`
	WalkingDistance string           `json:"walking_distance"`
	Segments        []transitSegment `json:"segments"`
}

type transitSegment struct {
	Walking struct {
		Distance string        `json:"distance"`
		Duration string        `json:"duration"`
		Steps    []walkingStep `json:"steps"`
	} `json:"walking"`
	Bus struct {
		BusLines []transitBusLine `json:"buslines"`
	} `json:"bus"`
}

type transitBusLine struct {
	Name      string `json:"name"`
	Distance  string `json:"distance"`
	Duration  string `json:"duration"`
	Polyline  string `json:"polyline"`
	Departure struct {
		Name string `json:"name"`
	} `json:"departure_stop"`
	Arrival struct {
		Name string `json:"name"`
	} `json:"arrival_stop"`
}

func buildTransitRoute(origin, destination routing.LocationPoint, source transitPlan) (routing.WalkingRoute, error) {
	duration, err := parseInt32(source.Duration)
	if err != nil {
		return routing.WalkingRoute{}, fmt.Errorf("%w: invalid AMap transit duration", routing.ErrProvider)
	}
	var totalDistance int32
	points := make([]routing.RoutePoint, 0)
	steps := make([]routing.Step, 0)
	for _, segment := range source.Segments {
		for _, walk := range segment.Walking.Steps {
			stepDistance, err := parseInt32(walk.Distance)
			if err != nil {
				return routing.WalkingRoute{}, fmt.Errorf("%w: invalid AMap transit walking distance", routing.ErrProvider)
			}
			stepDuration, err := parseOptionalInt32(walk.Duration)
			if err != nil {
				return routing.WalkingRoute{}, fmt.Errorf("%w: invalid AMap transit walking duration", routing.ErrProvider)
			}
			polyline, err := parsePolyline(walk.Polyline)
			if err != nil {
				return routing.WalkingRoute{}, fmt.Errorf("%w: invalid AMap transit walking polyline", routing.ErrProvider)
			}
			points = appendDistinct(points, polyline...)
			totalDistance += stepDistance
			steps = append(steps, routing.Step{Instruction: strings.TrimSpace(walk.Instruction), RoadName: strings.TrimSpace(walk.Road.String()), DistanceMeters: stepDistance, DurationSeconds: stepDuration})
		}
		for _, bus := range segment.Bus.BusLines {
			stepDistance, err := parseInt32(bus.Distance)
			if err != nil {
				return routing.WalkingRoute{}, fmt.Errorf("%w: invalid AMap transit line distance", routing.ErrProvider)
			}
			stepDuration, err := parseOptionalInt32(bus.Duration)
			if err != nil {
				return routing.WalkingRoute{}, fmt.Errorf("%w: invalid AMap transit line duration", routing.ErrProvider)
			}
			polyline, err := parsePolyline(bus.Polyline)
			if err != nil {
				return routing.WalkingRoute{}, fmt.Errorf("%w: invalid AMap transit line polyline", routing.ErrProvider)
			}
			points = appendDistinct(points, polyline...)
			totalDistance += stepDistance
			instruction := strings.TrimSpace(bus.Name)
			if bus.Departure.Name != "" || bus.Arrival.Name != "" {
				instruction = fmt.Sprintf("乘坐%s：%s 至 %s", instruction, strings.TrimSpace(bus.Departure.Name), strings.TrimSpace(bus.Arrival.Name))
			}
			steps = append(steps, routing.Step{Instruction: instruction, RoadName: strings.TrimSpace(bus.Name), DistanceMeters: stepDistance, DurationSeconds: stepDuration})
		}
	}
	if len(points) == 0 {
		return routing.WalkingRoute{}, routing.ErrNoRoute
	}
	return routing.WalkingRoute{Origin: origin, Destination: destination, DistanceMeters: totalDistance, DurationSeconds: duration, Polyline: points, Steps: steps, Provider: "amap", Mode: routing.RouteModeTransit}, nil
}

func parseOptionalInt32(value string) (int32, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	return parseInt32(value)
}
