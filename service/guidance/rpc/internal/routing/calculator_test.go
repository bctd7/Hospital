package routing

import (
	"context"
	"errors"
	"testing"
)

type providerStub struct {
	origin      LocationPoint
	destination LocationPoint
}

func (p *providerStub) CalculateWalkingRoute(_ context.Context, origin, destination LocationPoint) (WalkingRoute, error) {
	p.origin, p.destination = origin, destination
	return WalkingRoute{Origin: origin, Destination: destination, Provider: "stub"}, nil
}

func TestCalculatorNormalizesAndForwardsUnifiedLocationPoints(t *testing.T) {
	provider := &providerStub{}
	calculator, err := NewCalculator(provider)
	if err != nil {
		t.Fatal(err)
	}
	_, err = calculator.CalculateWalkingRoute(context.Background(),
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

func TestCalculatorRejectsInvalidCoordinates(t *testing.T) {
	calculator, _ := NewCalculator(&providerStub{})
	_, err := calculator.CalculateWalkingRoute(context.Background(),
		LocationPoint{Name: "A", Latitude: 91, Longitude: 121},
		LocationPoint{Name: "B", Latitude: 31, Longitude: 121},
	)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}
