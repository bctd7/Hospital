package config

import (
	"errors"
	"os"
	"strings"

	"github.com/zeromicro/go-zero/zrpc"
)

var ErrIncompleteProviderEnvironment = errors.New("incomplete guidance provider environment")

type Config struct {
	zrpc.RpcServerConf
	Environment string `json:",default=local"`
	MySQL       struct {
		DataSource string
	}
	AuthorizationRedis struct {
		Addr          string
		Password      string `json:",optional"`
		DB            int    `json:",default=0"`
		VersionPrefix string `json:",default=identity:authorization-version:"`
	}
	Token struct {
		Issuer                string `json:",default=hospital-identity"`
		Audience              string `json:",default=hospital-services"`
		AccessPublicKeyBase64 string
		AccessTTLSeconds      int64 `json:",default=900"`
	}
	AppointmentRPC zrpc.RpcClientConf
	AMap           struct {
		PlaceSearchEndpoint string `json:",default=https://restapi.amap.com/v5/place/text"`
		GeocodeEndpoint     string `json:",default=https://restapi.amap.com/v3/geocode/geo"`
		WalkingEndpoint     string `json:",default=https://restapi.amap.com/v3/direction/walking"`
		DrivingEndpoint     string `json:",default=https://restapi.amap.com/v3/direction/driving"`
		TransitEndpoint     string `json:",default=https://restapi.amap.com/v3/direction/transit/integrated"`
		WebServiceKey       string `json:",optional"`
		TimeoutMilliseconds int64  `json:",default=5000"`
	}
	LLM struct {
		Endpoint            string `json:",optional"`
		APIKey              string `json:",optional"`
		Model               string `json:",optional"`
		TimeoutMilliseconds int64  `json:",default=10000"`
	}
}

// BindProviderEnvironment 在配置文件加载后显式绑定第三方服务配置。
// 生产镜像不再依赖配置库对 ${VAR} 的隐式展开行为。
func BindProviderEnvironment(c *Config) error {
	if c == nil {
		return ErrIncompleteProviderEnvironment
	}
	bind := func(name string, target *string) {
		if value, exists := os.LookupEnv(name); exists {
			*target = strings.TrimSpace(value)
		}
	}
	bind("AMAP_WEB_SERVICE_KEY", &c.AMap.WebServiceKey)
	bind("GUIDANCE_LLM_ENDPOINT", &c.LLM.Endpoint)
	bind("GUIDANCE_LLM_API_KEY", &c.LLM.APIKey)
	bind("GUIDANCE_LLM_MODEL", &c.LLM.Model)

	llmValues := []string{c.LLM.Endpoint, c.LLM.APIKey, c.LLM.Model}
	configured := 0
	for _, value := range llmValues {
		if strings.TrimSpace(value) != "" {
			configured++
		}
	}
	if configured != 0 && configured != len(llmValues) {
		return ErrIncompleteProviderEnvironment
	}
	return nil
}
