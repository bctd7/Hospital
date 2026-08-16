package amap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"hospital/service/guidance/rpc/internal/routing"
)

const (
	defaultPlaceSearchEndpoint = "https://restapi.amap.com/v5/place/text"
	defaultGeocodeEndpoint     = "https://restapi.amap.com/v3/geocode/geo"
	defaultWalkingEndpoint     = "https://restapi.amap.com/v3/direction/walking"
)

type Config struct {
	PlaceSearchEndpoint string
	GeocodeEndpoint     string
	WalkingEndpoint     string
	WebServiceKey       string
	Timeout             time.Duration
}

type Client struct {
	placeSearchEndpoint string
	geocodeEndpoint     string
	walkingEndpoint     string
	webServiceKey       string
	httpClient          *http.Client
}

func New(config Config) *Client {
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &Client{
		placeSearchEndpoint: endpointOrDefault(config.PlaceSearchEndpoint, defaultPlaceSearchEndpoint),
		geocodeEndpoint:     endpointOrDefault(config.GeocodeEndpoint, defaultGeocodeEndpoint),
		walkingEndpoint:     endpointOrDefault(config.WalkingEndpoint, defaultWalkingEndpoint),
		webServiceKey:       strings.TrimSpace(config.WebServiceKey),
		httpClient:          &http.Client{Timeout: timeout},
	}
}

func endpointOrDefault(value, fallback string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return fallback
}

func (c *Client) request(ctx context.Context, endpoint string, query url.Values, target any) error {
	if c.webServiceKey == "" {
		return routing.ErrUnavailable
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("%w: invalid AMap endpoint", routing.ErrProvider)
	}
	query.Set("key", c.webServiceKey)
	query.Set("output", "json")
	parsed.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return fmt.Errorf("%w: create AMap request", routing.ErrProvider)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return fmt.Errorf("%w: AMap request failed", routing.ErrUnavailable)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, response.Body)
		return fmt.Errorf("%w: AMap http status %d", routing.ErrUnavailable, response.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(target); err != nil {
		return fmt.Errorf("%w: decode AMap response", routing.ErrProvider)
	}
	return nil
}

type responseStatus struct {
	Status   string `json:"status"`
	Info     string `json:"info"`
	InfoCode string `json:"infocode"`
}

func (s responseStatus) validate() error {
	if s.Status == "1" {
		return nil
	}
	return fmt.Errorf("%w: AMap infocode %s (%s)", routing.ErrUnavailable, strings.TrimSpace(s.InfoCode), strings.TrimSpace(s.Info))
}
