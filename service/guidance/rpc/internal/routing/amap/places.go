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
	query.Set("page_size", strconv.Itoa(int(input.Limit)))
	query.Set("page_num", "1")
	query.Set("show_fields", "navi")
	if input.City != "" {
		query.Set("region", input.City)
		query.Set("city_limit", "true")
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
		latitude, longitude, err := parseCoordinate(source.routeLocation())
		if err != nil || strings.TrimSpace(source.Name.String()) == "" {
			continue
		}
		places = append(places, routing.LocationPoint{
			Name: strings.TrimSpace(source.Name.String()), Address: placeAddress(source),
			Latitude: latitude, Longitude: longitude, ProviderPlaceID: strings.TrimSpace(source.ID.String()),
		})
	}
	if hasExactPlace(places, input.Keyword) {
		return places, nil
	}
	exact, err := c.geocode(ctx, input.Keyword, input.City)
	if err != nil {
		if len(places) > 0 {
			return places, nil
		}
		return nil, err
	}
	if exact != nil && !containsCoordinate(places, *exact) {
		places = append([]routing.LocationPoint{*exact}, places...)
		if len(places) > int(input.Limit) {
			places = places[:input.Limit]
		}
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
	Navi     struct {
		Entrance flexibleString `json:"entr_location"`
	} `json:"navi"`
}

func (p placePOI) routeLocation() string {
	if entrance := strings.TrimSpace(p.Navi.Entrance.String()); entrance != "" {
		return entrance
	}
	return p.Location.String()
}

type geocodeResponse struct {
	responseStatus
	Geocodes []struct {
		FormattedAddress flexibleString `json:"formatted_address"`
		Location         flexibleString `json:"location"`
	} `json:"geocodes"`
}

func (c *Client) geocode(ctx context.Context, keyword, city string) (*routing.LocationPoint, error) {
	query := url.Values{}
	query.Set("address", keyword)
	if city != "" {
		query.Set("city", city)
	}
	var payload geocodeResponse
	if err := c.request(ctx, c.geocodeEndpoint, query, &payload); err != nil {
		return nil, err
	}
	if err := payload.responseStatus.validate(); err != nil {
		return nil, err
	}
	for _, source := range payload.Geocodes {
		latitude, longitude, err := parseCoordinate(source.Location.String())
		if err != nil {
			continue
		}
		return &routing.LocationPoint{
			Name: keyword, Address: strings.TrimSpace(source.FormattedAddress.String()),
			Latitude: latitude, Longitude: longitude,
		}, nil
	}
	return nil, nil
}

func hasExactPlace(places []routing.LocationPoint, keyword string) bool {
	normalizedKeyword := normalizePlaceName(keyword)
	for _, place := range places {
		if normalizePlaceName(place.Name) == normalizedKeyword {
			return true
		}
	}
	return false
}

func normalizePlaceName(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), "")
}

func containsCoordinate(places []routing.LocationPoint, candidate routing.LocationPoint) bool {
	for _, place := range places {
		if place.Latitude == candidate.Latitude && place.Longitude == candidate.Longitude {
			return true
		}
	}
	return false
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
