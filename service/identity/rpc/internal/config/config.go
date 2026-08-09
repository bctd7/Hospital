package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	MySQL struct {
		DataSource string
	}
	SessionRedis struct {
		Addr     string
		Password string `json:",optional"`
		DB       int    `json:",default=0"`
		Prefix   string `json:",default=identity:refresh:"`
	}
	Token struct {
		Issuer                 string `json:",default=hospital-identity"`
		Audience               string `json:",default=hospital-services"`
		AccessPrivateKeyBase64 string
		AccessPublicKeyBase64  string
		AccessTTLSeconds       int64 `json:",default=900"`
		RefreshTTLSeconds      int64 `json:",default=2592000"`
	}
}
