package baidumap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"hospital/service/guidance/rpc/internal/routing"
)

func TestClientCalculatesGCJ02WalkingRoute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		if request.URL.Path != "/direction/v2/walking" || query.Get("coord_type") != "gcj02" || query.Get("ret_coordtype") != "gcj02" {
			t.Fatalf("unexpected route request: %s", request.URL.String())
		}
		if query.Get("origin") != "31.200000,121.400000" || query.Get("destination") != "31.210000,121.410000" {
			t.Fatalf("unexpected coordinates: %s", request.URL.RawQuery)
		}
		if query.Get("ak") != "test-ak" || query.Get("sn") == "" || query.Get("timestamp") == "" {
			t.Fatalf("missing signed credentials: %s", request.URL.RawQuery)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
          "status":0,"message":"ok","result":{"routes":[{
            "distance":820,"duration":640,"steps":[
              {"distance":320,"duration":240,"instructions":"向<b>东</b>步行","name":"院区路","path":"121.400000,31.200000;121.405000,31.205000"},
              {"distance":500,"duration":400,"instructions":"到达终点","name":"楼前路","path":"121.405000,31.205000;121.410000,31.210000"}
            ]
          }]}}
        `))
	}))
	defer server.Close()

	client := New(Config{Endpoint: server.URL + "/direction/v2/walking", AccessKey: "test-ak", SecurityKey: "test-sk"})
	result, err := client.CalculateWalkingRoute(context.Background(),
		routing.LocationPoint{Name: "A", Latitude: 31.2, Longitude: 121.4},
		routing.LocationPoint{Name: "B", Latitude: 31.21, Longitude: 121.41},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.DistanceMeters != 820 || result.DurationSeconds != 640 || result.Provider != "baidu" {
		t.Fatalf("unexpected summary: %#v", result)
	}
	if len(result.Polyline) != 3 || len(result.Steps) != 2 || result.Steps[0].Instruction != "向东步行" {
		t.Fatalf("unexpected route details: %#v", result)
	}
}

func TestClientReturnsUnavailableWithoutServerCredentials(t *testing.T) {
	client := New(Config{})
	_, err := client.CalculateWalkingRoute(context.Background(),
		routing.LocationPoint{Name: "A", Latitude: 31.2, Longitude: 121.4},
		routing.LocationPoint{Name: "B", Latitude: 31.21, Longitude: 121.41},
	)
	if !errors.Is(err, routing.ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
}
