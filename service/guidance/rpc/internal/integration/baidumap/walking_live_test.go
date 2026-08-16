package baidumap

import (
	"context"
	"os"
	"testing"
	"time"

	"hospital/service/guidance/rpc/internal/routing"
)

// 该测试默认跳过；本地显式配置百度服务端密钥后，可用于验证真实配额、签名和返回格式。
func TestLiveClientCalculatesWalkingRoute(t *testing.T) {
	accessKey := os.Getenv("BAIDU_MAP_SERVER_AK")
	securityKey := os.Getenv("BAIDU_MAP_SERVER_SK")
	if accessKey == "" || securityKey == "" {
		t.Skip("Baidu server credentials are not configured")
	}
	client := New(Config{AccessKey: accessKey, SecurityKey: securityKey, Timeout: 8 * time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := client.CalculateWalkingRoute(ctx,
		routing.LocationPoint{Name: "人民广场", Latitude: 31.230391, Longitude: 121.473701},
		routing.LocationPoint{Name: "上海博物馆", Latitude: 31.228917, Longitude: 121.475188},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.DistanceMeters <= 0 || result.DurationSeconds <= 0 || len(result.Polyline) < 2 {
		t.Fatalf("unexpected live route summary: distance=%d duration=%d points=%d",
			result.DistanceMeters, result.DurationSeconds, len(result.Polyline))
	}
}
