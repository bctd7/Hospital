package config

import "github.com/zeromicro/go-zero/zrpc"

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
		WebServiceKey       string `json:",optional,env=AMAP_WEB_SERVICE_KEY"`
		TimeoutMilliseconds int64  `json:",default=5000"`
	}
	LLM struct {
		Endpoint            string `json:",optional,env=GUIDANCE_LLM_ENDPOINT"`
		APIKey              string `json:",optional,env=GUIDANCE_LLM_API_KEY"`
		Model               string `json:",optional,env=GUIDANCE_LLM_MODEL"`
		TimeoutMilliseconds int64  `json:",default=10000"`
	}
}
