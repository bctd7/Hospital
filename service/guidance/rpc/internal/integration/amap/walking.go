package amap

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"hospital/service/guidance/rpc/internal/routing"
)

func (c *Client) CalculateWalkingRoute(ctx context.Context, origin, destination routing.LocationPoint) (routing.WalkingRoute, error) {
	query := url.Values{}
	query.Set("origin", coordinate(origin))
	query.Set("destination", coordinate(destination))
	if origin.ProviderPlaceID != "" {
		query.Set("origin_id", origin.ProviderPlaceID)
	}
	if destination.ProviderPlaceID != "" {
		query.Set("destination_id", destination.ProviderPlaceID)
	}
	var payload walkingResponse
	if err := c.request(ctx, c.walkingEndpoint, query, &payload); err != nil {
		return routing.WalkingRoute{}, err
	}
	if err := payload.responseStatus.validate(); err != nil {
		return routing.WalkingRoute{}, err
	}
	if len(payload.Route.Paths) == 0 {
		return routing.WalkingRoute{}, routing.ErrNoRoute
	}
	return buildRoute(origin, destination, payload.Route.Paths[0])
}

func coordinate(point routing.LocationPoint) string {
	return strconv.FormatFloat(point.Longitude, 'f', 6, 64) + "," + strconv.FormatFloat(point.Latitude, 'f', 6, 64)
}

type walkingResponse struct {
	responseStatus
	Route struct {
		Paths []walkingPath `json:"paths"`
	} `json:"route"`
}

type walkingPath struct {
	Distance string        `json:"distance"`
	Duration string        `json:"duration"`
	Steps    []walkingStep `json:"steps"`
}

type walkingStep struct {
	Instruction string `json:"instruction"`
	Road        string `json:"road"`
	Distance    string `json:"distance"`
	Duration    string `json:"duration"`
	Polyline    string `json:"polyline"`
}

func buildRoute(origin, destination routing.LocationPoint, source walkingPath) (routing.WalkingRoute, error) {
	distance, err := parseInt32(source.Distance)
	if err != nil {
		return routing.WalkingRoute{}, fmt.Errorf("%w: invalid AMap route distance", routing.ErrProvider)
	}
	duration, err := parseInt32(source.Duration)
	if err != nil {
		return routing.WalkingRoute{}, fmt.Errorf("%w: invalid AMap route duration", routing.ErrProvider)
	}
	points := make([]routing.RoutePoint, 0)
	steps := make([]routing.Step, 0, len(source.Steps))
	for _, sourceStep := range source.Steps {
		parsed, err := parsePolyline(sourceStep.Polyline)
		if err != nil {
			return routing.WalkingRoute{}, fmt.Errorf("%w: invalid AMap route polyline", routing.ErrProvider)
		}
		points = appendDistinct(points, parsed...)
		stepDistance, err := parseInt32(sourceStep.Distance)
		if err != nil {
			return routing.WalkingRoute{}, fmt.Errorf("%w: invalid AMap step distance", routing.ErrProvider)
		}
		stepDuration, err := parseInt32(sourceStep.Duration)
		if err != nil {
			return routing.WalkingRoute{}, fmt.Errorf("%w: invalid AMap step duration", routing.ErrProvider)
		}
		steps = append(steps, routing.Step{
			Instruction: strings.TrimSpace(sourceStep.Instruction), RoadName: strings.TrimSpace(sourceStep.Road),
			DistanceMeters: stepDistance, DurationSeconds: stepDuration,
		})
	}
	if len(points) == 0 {
		return routing.WalkingRoute{}, routing.ErrNoRoute
	}
	return routing.WalkingRoute{
		Origin: origin, Destination: destination, DistanceMeters: distance, DurationSeconds: duration,
		Polyline: points, Steps: steps, Provider: "amap",
	}, nil
}

func parseCoordinate(value string) (latitude, longitude float64, err error) {
	parts := strings.Split(strings.TrimSpace(value), ",")
	if len(parts) != 2 {
		return 0, 0, routing.ErrProvider
	}
	longitude, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, routing.ErrProvider
	}
	latitude, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, routing.ErrProvider
	}
	return latitude, longitude, nil
}

func parsePolyline(value string) ([]routing.RoutePoint, error) {
	segments := strings.Split(strings.TrimSpace(value), ";")
	points := make([]routing.RoutePoint, 0, len(segments))
	for _, segment := range segments {
		if strings.TrimSpace(segment) == "" {
			continue
		}
		latitude, longitude, err := parseCoordinate(segment)
		if err != nil {
			return nil, err
		}
		points = append(points, routing.RoutePoint{Latitude: latitude, Longitude: longitude})
	}
	return points, nil
}

func parseInt32(value string) (int32, error) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
	return int32(parsed), err
}

func appendDistinct(target []routing.RoutePoint, values ...routing.RoutePoint) []routing.RoutePoint {
	for _, value := range values {
		if len(target) == 0 || target[len(target)-1] != value {
			target = append(target, value)
		}
	}
	return target
}
