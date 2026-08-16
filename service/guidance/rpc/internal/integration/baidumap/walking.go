package baidumap

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"hospital/service/guidance/rpc/internal/routing"
)

const defaultWalkingEndpoint = "https://api.map.baidu.com/direction/v2/walking"

var instructionTagPattern = regexp.MustCompile(`<[^>]+>`)

type Config struct {
	Endpoint    string
	AccessKey   string
	SecurityKey string
	Timeout     time.Duration
}

type Client struct {
	endpoint    string
	accessKey   string
	securityKey string
	httpClient  *http.Client
}

func New(config Config) *Client {
	endpoint := strings.TrimSpace(config.Endpoint)
	if endpoint == "" {
		endpoint = defaultWalkingEndpoint
	}
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &Client{
		endpoint: endpoint, accessKey: strings.TrimSpace(config.AccessKey),
		securityKey: strings.TrimSpace(config.SecurityKey), httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *Client) CalculateWalkingRoute(ctx context.Context, origin, destination routing.LocationPoint) (routing.WalkingRoute, error) {
	if c.accessKey == "" || c.securityKey == "" {
		return routing.WalkingRoute{}, routing.ErrUnavailable
	}
	target, err := url.Parse(c.endpoint)
	if err != nil {
		return routing.WalkingRoute{}, fmt.Errorf("%w: invalid endpoint", routing.ErrProvider)
	}
	query := url.Values{}
	query.Set("ak", c.accessKey)
	query.Set("coord_type", "gcj02")
	query.Set("destination", coordinate(destination))
	if destination.ProviderPlaceID != "" {
		query.Set("destination_uid", destination.ProviderPlaceID)
	}
	query.Set("origin", coordinate(origin))
	if origin.ProviderPlaceID != "" {
		query.Set("origin_uid", origin.ProviderPlaceID)
	}
	query.Set("output", "json")
	query.Set("ret_coordtype", "gcj02")
	query.Set("timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	query.Set("sn", calculateSN(target.EscapedPath(), query.Encode(), c.securityKey))
	target.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return routing.WalkingRoute{}, fmt.Errorf("%w: create request", routing.ErrProvider)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return routing.WalkingRoute{}, err
		}
		return routing.WalkingRoute{}, fmt.Errorf("%w: request failed", routing.ErrUnavailable)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, response.Body)
		return routing.WalkingRoute{}, fmt.Errorf("%w: http status %d", routing.ErrUnavailable, response.StatusCode)
	}
	var payload walkingResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&payload); err != nil {
		return routing.WalkingRoute{}, fmt.Errorf("%w: decode response", routing.ErrProvider)
	}
	if payload.Status == 2001 {
		return routing.WalkingRoute{}, routing.ErrNoRoute
	}
	if payload.Status != 0 {
		return routing.WalkingRoute{}, fmt.Errorf("%w: baidu status %d (%s)", routing.ErrProvider, payload.Status, payload.Message)
	}
	if len(payload.Result.Routes) == 0 {
		return routing.WalkingRoute{}, routing.ErrNoRoute
	}
	return buildRoute(origin, destination, payload.Result.Routes[0])
}

func coordinate(point routing.LocationPoint) string {
	return strconv.FormatFloat(point.Latitude, 'f', 6, 64) + "," + strconv.FormatFloat(point.Longitude, 'f', 6, 64)
}

func calculateSN(path, encodedQuery, securityKey string) string {
	encoded := url.QueryEscape(path + "?" + encodedQuery + securityKey)
	digest := md5.Sum([]byte(encoded))
	return hex.EncodeToString(digest[:])
}

type walkingResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Result  struct {
		Routes []walkingRoute `json:"routes"`
	} `json:"result"`
}

type walkingRoute struct {
	Distance int `json:"distance"`
	Duration int `json:"duration"`
	Steps    []struct {
		Distance     int    `json:"distance"`
		Duration     int    `json:"duration"`
		Instructions string `json:"instructions"`
		Name         string `json:"name"`
		Path         string `json:"path"`
	} `json:"steps"`
}

func buildRoute(origin, destination routing.LocationPoint, source walkingRoute) (routing.WalkingRoute, error) {
	points := make([]routing.RoutePoint, 0)
	steps := make([]routing.Step, 0, len(source.Steps))
	for _, sourceStep := range source.Steps {
		parsed, err := parsePath(sourceStep.Path)
		if err != nil {
			return routing.WalkingRoute{}, fmt.Errorf("%w: invalid route path", routing.ErrProvider)
		}
		points = appendDistinct(points, parsed...)
		steps = append(steps, routing.Step{
			Instruction: cleanInstruction(sourceStep.Instructions), RoadName: strings.TrimSpace(sourceStep.Name),
			DistanceMeters: int32(sourceStep.Distance), DurationSeconds: int32(sourceStep.Duration),
		})
	}
	if len(points) == 0 {
		return routing.WalkingRoute{}, routing.ErrNoRoute
	}
	return routing.WalkingRoute{
		Origin: origin, Destination: destination, DistanceMeters: int32(source.Distance),
		DurationSeconds: int32(source.Duration), Polyline: points, Steps: steps, Provider: "baidu",
	}, nil
}

func parsePath(value string) ([]routing.RoutePoint, error) {
	segments := strings.Split(strings.TrimSpace(value), ";")
	points := make([]routing.RoutePoint, 0, len(segments))
	for _, segment := range segments {
		if strings.TrimSpace(segment) == "" {
			continue
		}
		parts := strings.Split(segment, ",")
		if len(parts) != 2 {
			return nil, routing.ErrProvider
		}
		longitude, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			return nil, routing.ErrProvider
		}
		latitude, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			return nil, routing.ErrProvider
		}
		points = append(points, routing.RoutePoint{Latitude: latitude, Longitude: longitude})
	}
	return points, nil
}

func appendDistinct(target []routing.RoutePoint, values ...routing.RoutePoint) []routing.RoutePoint {
	for _, value := range values {
		if len(target) == 0 || target[len(target)-1] != value {
			target = append(target, value)
		}
	}
	return target
}

func cleanInstruction(value string) string {
	return strings.TrimSpace(html.UnescapeString(instructionTagPattern.ReplaceAllString(value, "")))
}
