package amap

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"hospital/service/guidance/rpc/internal/routing"
)

// TestLiveClient 使用真实高德 Web 服务验证地点搜索与步行路线。
// 常规测试环境没有配置 Key 时自动跳过，避免把外部网络作为项目构建前提。
func TestLiveClient(t *testing.T) {
	key := strings.TrimSpace(os.Getenv("AMAP_WEB_SERVICE_KEY"))
	if key == "" {
		t.Skip("AMAP_WEB_SERVICE_KEY is not configured")
	}
	client := New(Config{WebServiceKey: key, Timeout: 10 * time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	origins, err := client.SearchPlaces(ctx, routing.PlaceSearchInput{Keyword: "上海市第一人民医院北部", City: "上海市", Limit: 3})
	if err != nil {
		t.Fatalf("search origin: %v", err)
	}
	destinations, err := client.SearchPlaces(ctx, routing.PlaceSearchInput{Keyword: "上海市第十人民医院", City: "上海市", Limit: 3})
	if err != nil {
		t.Fatalf("search destination: %v", err)
	}
	if len(origins) == 0 || len(destinations) == 0 {
		t.Fatalf("unexpected empty places: origins=%d destinations=%d", len(origins), len(destinations))
	}

	route, err := client.CalculateWalkingRoute(ctx, origins[0], destinations[0])
	if err != nil {
		t.Fatalf("calculate walking route: %v", err)
	}
	if route.Provider != "amap" || route.DistanceMeters <= 0 || route.DurationSeconds <= 0 || len(route.Polyline) < 2 || len(route.Steps) == 0 {
		t.Fatalf("unexpected live route: %#v", route)
	}
}
