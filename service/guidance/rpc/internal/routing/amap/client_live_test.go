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

	buildingOne, err := client.SearchPlaces(ctx, routing.PlaceSearchInput{
		Keyword: "上海市第二人民医院1号楼", City: "上海市", Limit: 5,
	})
	if err != nil {
		t.Fatalf("search building one: %v", err)
	}
	// 精确楼栋搜索会依次调用地点检索和地理编码；错开两组请求，避免测试本身触发免费额度的瞬时 QPS 限制。
	time.Sleep(time.Second)
	buildingTwo, err := client.SearchPlaces(ctx, routing.PlaceSearchInput{
		Keyword: "上海第二人民医院2号楼", City: "上海市", Limit: 5,
	})
	if err != nil {
		t.Fatalf("search building two: %v", err)
	}
	if len(buildingOne) == 0 || len(buildingTwo) == 0 ||
		buildingOne[0].Name != "上海市第二人民医院1号楼" || buildingTwo[0].Name != "上海第二人民医院2号楼" {
		t.Fatalf("building geocode fallback missing: one=%#v two=%#v", buildingOne, buildingTwo)
	}
	buildingRoute, err := client.CalculateWalkingRoute(ctx, buildingOne[0], buildingTwo[0])
	if err != nil {
		t.Fatalf("calculate building walking route: %v", err)
	}
	if buildingRoute.DistanceMeters <= 0 || buildingRoute.DurationSeconds <= 0 || len(buildingRoute.Polyline) < 2 {
		t.Fatalf("unexpected building route: %#v", buildingRoute)
	}
	t.Logf(
		"1号楼 %f,%f -> 2号楼 %f,%f：%d 米，%d 秒",
		buildingOne[0].Longitude,
		buildingOne[0].Latitude,
		buildingTwo[0].Longitude,
		buildingTwo[0].Latitude,
		buildingRoute.DistanceMeters,
		buildingRoute.DurationSeconds,
	)
}
