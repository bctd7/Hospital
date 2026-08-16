package routing

import (
	"context"
	"errors"
	"testing"
)

type providerStub struct {
	searchInput PlaceSearchInput
	origin      LocationPoint
	destination LocationPoint
	routeCalls  int
}

func (p *providerStub) SearchPlaces(_ context.Context, input PlaceSearchInput) ([]LocationPoint, error) {
	p.searchInput = input
	return []LocationPoint{{Name: "123号楼", Latitude: 31.1, Longitude: 121.1}}, nil
}

func (p *providerStub) CalculateWalkingRoute(_ context.Context, origin, destination LocationPoint) (WalkingRoute, error) {
	p.routeCalls++
	p.origin, p.destination = origin, destination
	return WalkingRoute{Origin: origin, Destination: destination, Provider: "stub"}, nil
}

func TestManagerNormalizesPlaceSearch(t *testing.T) {
	provider := &providerStub{}
	manager, err := NewManager(provider)
	if err != nil {
		t.Fatal(err)
	}
	places, err := manager.SearchPlaces(context.Background(), PlaceSearchInput{
		Keyword: " 123号楼 ", City: " 上海市 ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(places) != 1 || provider.searchInput.Keyword != "123号楼" ||
		provider.searchInput.City != "上海市" || provider.searchInput.Limit != 10 {
		t.Fatalf("unexpected normalized search: %#v %#v", provider.searchInput, places)
	}
}

func TestManagerRejectsInvalidPlaceSearch(t *testing.T) {
	manager, _ := NewManager(&providerStub{})
	for _, input := range []PlaceSearchInput{{}, {Keyword: "A", Limit: 21}, {Keyword: "A", Limit: -1}} {
		if _, err := manager.SearchPlaces(context.Background(), input); !errors.Is(err, ErrInvalid) {
			t.Fatalf("expected ErrInvalid for %#v, got %v", input, err)
		}
	}
}

func TestManagerNormalizesAndForwardsRoute(t *testing.T) {
	provider := &providerStub{}
	manager, err := NewManager(provider)
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.CalculateWalkingRoute(context.Background(),
		LocationPoint{Name: "  出发地  ", Address: "  地址 A ", Latitude: 31.2, Longitude: 121.4},
		LocationPoint{Name: " 一号楼 ", Latitude: 31.21, Longitude: 121.41},
	)
	if err != nil {
		t.Fatal(err)
	}
	if provider.origin.Name != "出发地" || provider.origin.Address != "地址 A" || provider.destination.Name != "一号楼" {
		t.Fatalf("unexpected normalized points: %#v %#v", provider.origin, provider.destination)
	}
}

func TestManagerHandlesSameLocationWithoutCallingProvider(t *testing.T) {
	provider := &providerStub{}
	manager, err := NewManager(provider)
	if err != nil {
		t.Fatal(err)
	}
	result, err := manager.CalculateWalkingRoute(context.Background(),
		LocationPoint{Name: "入口东侧", Latitude: 31.40527, Longitude: 121.48941},
		LocationPoint{Name: "入口", Latitude: 31.40527, Longitude: 121.48941},
	)
	if err != nil {
		t.Fatal(err)
	}
	if provider.routeCalls != 0 {
		t.Fatalf("same location should not call route provider, got %d calls", provider.routeCalls)
	}
	if result.Provider != "local" || len(result.Polyline) != 2 || result.DistanceMeters != 0 || result.DurationSeconds != 0 {
		t.Fatalf("unexpected same-location route: %#v", result)
	}
}

func TestManagerRejectsInvalidCoordinates(t *testing.T) {
	manager, _ := NewManager(&providerStub{})
	_, err := manager.CalculateWalkingRoute(context.Background(),
		LocationPoint{Name: "A", Latitude: 91, Longitude: 121},
		LocationPoint{Name: "B", Latitude: 31, Longitude: 121},
	)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}
