package amap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hospital/service/guidance/rpc/internal/routing"
)

func TestClientSearchesPlacesAndCalculatesWalkingRoute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		if query.Get("key") != "test-key" || query.Get("output") != "json" {
			t.Fatalf("missing common AMap query: %s", request.URL.RawQuery)
		}
		switch request.URL.Path {
		case "/place":
			if query.Get("keywords") != "123号楼" || query.Get("city") != "上海市" || query.Get("citylimit") != "true" {
				t.Fatalf("unexpected place query: %s", request.URL.RawQuery)
			}
			_, _ = writer.Write([]byte(`{
                  "status":"1","info":"OK","infocode":"10000",
                  "pois":[{"id":"B001","name":"123号楼","address":"友谊路1号","pname":"上海市","cityname":"上海市","adname":"宝山区","location":"121.489410,31.405270"}]
                }`))
		case "/walking":
			if query.Get("origin") != "121.400000,31.200000" || query.Get("destination") != "121.410000,31.210000" ||
				query.Get("origin_id") != "O1" || query.Get("destination_id") != "D1" {
				t.Fatalf("unexpected walking query: %s", request.URL.RawQuery)
			}
			_, _ = writer.Write([]byte(`{
                  "status":"1","info":"OK","infocode":"10000",
                  "route":{"paths":[{"distance":"820","duration":"640","steps":[
                    {"instruction":"向东步行","road":"院区路","distance":"320","duration":"240","polyline":"121.400000,31.200000;121.405000,31.205000"},
                    {"instruction":"到达终点","road":"楼前路","distance":"500","duration":"400","polyline":"121.405000,31.205000;121.410000,31.210000"}
                  ]}]}
                }`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := New(Config{
		PlaceSearchEndpoint: server.URL + "/place", WalkingEndpoint: server.URL + "/walking",
		WebServiceKey: "test-key", Timeout: time.Second,
	})
	places, err := client.SearchPlaces(context.Background(), routing.PlaceSearchInput{Keyword: "123号楼", City: "上海市", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(places) != 1 || places[0].Name != "123号楼" || places[0].ProviderPlaceID != "B001" ||
		places[0].Longitude != 121.489410 || places[0].Latitude != 31.405270 {
		t.Fatalf("unexpected places: %#v", places)
	}

	route, err := client.CalculateWalkingRoute(context.Background(),
		routing.LocationPoint{Name: "A", Latitude: 31.2, Longitude: 121.4, ProviderPlaceID: "O1"},
		routing.LocationPoint{Name: "B", Latitude: 31.21, Longitude: 121.41, ProviderPlaceID: "D1"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if route.Provider != "amap" || route.DistanceMeters != 820 || route.DurationSeconds != 640 || len(route.Polyline) != 3 || len(route.Steps) != 2 {
		t.Fatalf("unexpected route: %#v", route)
	}
}

func TestClientMapsAMapFailureToUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"status":"0","info":"INVALID_USER_KEY","infocode":"10001"}`))
	}))
	defer server.Close()
	client := New(Config{PlaceSearchEndpoint: server.URL, WebServiceKey: "bad-key", Timeout: time.Second})
	_, err := client.SearchPlaces(context.Background(), routing.PlaceSearchInput{Keyword: "医院", Limit: 10})
	if !errors.Is(err, routing.ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
}

func TestClientRequiresWebServiceKey(t *testing.T) {
	client := New(Config{})
	_, err := client.SearchPlaces(context.Background(), routing.PlaceSearchInput{Keyword: "医院", Limit: 10})
	if !errors.Is(err, routing.ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
}
