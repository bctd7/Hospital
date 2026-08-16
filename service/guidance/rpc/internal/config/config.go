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
}
