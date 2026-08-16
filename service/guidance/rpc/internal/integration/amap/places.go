package amap

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"hospital/service/guidance/rpc/internal/routing"
)

func (c *Client) SearchPlaces(ctx context.Context, input routing.PlaceSearchInput) ([]routing.LocationPoint, error) {
	query := url.Values{}
	query.Set("keywords", input.Keyword)
	query.Set("offset", strconv.Itoa(int(input.Limit)))
	query.Set("page", "1")
	query.Set("extensions", "base")
	if input.City != "" {
		query.Set("city", input.City)
		query.Set("citylimit", "true")
	}
	var payload placeSearchResponse
	if err := c.request(ctx, c.placeSearchEndpoint, query, &payload); err != nil {
		return nil, err
	}
	if err := payload.responseStatus.validate(); err != nil {
		return nil, err
	}
	places := make([]routing.LocationPoint, 0, len(payload.POIs))
	for _, source := range payload.POIs {
		latitude, longitude, err := parseCoordinate(source.Location.String())
		if err != nil || strings.TrimSpace(source.Name.String()) == "" {
			continue
		}
		places = append(places, routing.LocationPoint{
			Name: strings.TrimSpace(source.Name.String()), Address: placeAddress(source),
			Latitude: latitude, Longitude: longitude, ProviderPlaceID: strings.TrimSpace(source.ID.String()),
		})
	}
	return places, nil
}

type placeSearchResponse struct {
	responseStatus
	POIs []placePOI `json:"pois"`
}

type placePOI struct {
	ID       flexibleString `json:"id"`
	Name     flexibleString `json:"name"`
	Address  flexibleString `json:"address"`
	Province flexibleString `json:"pname"`
	City     flexibleString `json:"cityname"`
	District flexibleString `json:"adname"`
	Location flexibleString `json:"location"`
}

type flexibleString string

func (s *flexibleString) UnmarshalJSON(value []byte) error {
	var text string
	if err := json.Unmarshal(value, &text); err == nil {
		*s = flexibleString(text)
		return nil
	}
	var values []string
	if err := json.Unmarshal(value, &values); err == nil {
		*s = flexibleString(strings.Join(values, ""))
		return nil
	}
	if string(value) == "null" {
		*s = ""
		return nil
	}
	return fmt.Errorf("unsupported AMap string value")
}

func (s flexibleString) String() string {
	return string(s)
}

func placeAddress(source placePOI) string {
	parts := []string{source.Province.String(), source.City.String(), source.District.String(), source.Address.String()}
	var builder strings.Builder
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || strings.Contains(builder.String(), part) {
			continue
		}
		builder.WriteString(part)
	}
	return builder.String()
}
