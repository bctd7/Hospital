package routing

import (
	"context"
	"strings"
)

const (
	defaultPlaceLimit = 10
	maximumPlaceLimit = 20
)

type PlaceSearchInput struct {
	Keyword string
	City    string
	Limit   int32
}

type PlaceProvider interface {
	SearchPlaces(context.Context, PlaceSearchInput) ([]LocationPoint, error)
}

// PlaceFinder 负责统一地点搜索参数，地图供应商只处理外部 API 细节。
type PlaceFinder struct {
	provider PlaceProvider
}

func NewPlaceFinder(provider PlaceProvider) (*PlaceFinder, error) {
	if provider == nil {
		return nil, ErrInvalid
	}
	return &PlaceFinder{provider: provider}, nil
}

func (f *PlaceFinder) Search(ctx context.Context, input PlaceSearchInput) ([]LocationPoint, error) {
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
	return f.provider.SearchPlaces(ctx, input)
}
