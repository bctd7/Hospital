package routing

import (
	"context"
	"errors"
	"testing"
)

type placeProviderStub struct {
	input PlaceSearchInput
}

func (p *placeProviderStub) SearchPlaces(_ context.Context, input PlaceSearchInput) ([]LocationPoint, error) {
	p.input = input
	return []LocationPoint{{Name: "123号楼", Latitude: 31.1, Longitude: 121.1}}, nil
}

func TestPlaceFinderNormalizesSearch(t *testing.T) {
	provider := &placeProviderStub{}
	finder, err := NewPlaceFinder(provider)
	if err != nil {
		t.Fatal(err)
	}
	places, err := finder.Search(context.Background(), PlaceSearchInput{Keyword: " 123号楼 ", City: " 上海市 "})
	if err != nil {
		t.Fatal(err)
	}
	if len(places) != 1 || provider.input.Keyword != "123号楼" || provider.input.City != "上海市" || provider.input.Limit != 10 {
		t.Fatalf("unexpected normalized search: %#v %#v", provider.input, places)
	}
}

func TestPlaceFinderRejectsInvalidInput(t *testing.T) {
	finder, _ := NewPlaceFinder(&placeProviderStub{})
	for _, input := range []PlaceSearchInput{{}, {Keyword: "A", Limit: 21}, {Keyword: "A", Limit: -1}} {
		if _, err := finder.Search(context.Background(), input); !errors.Is(err, ErrInvalid) {
			t.Fatalf("expected ErrInvalid for %#v, got %v", input, err)
		}
	}
}
