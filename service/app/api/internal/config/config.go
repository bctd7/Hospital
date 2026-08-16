// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	Environment        string `json:",default=local"`
	IdentityRPC        zrpc.RpcClientConf
	AppointmentRPC     zrpc.RpcClientConf
	GuidanceRPC        zrpc.RpcClientConf
	AuthorizationRedis struct {
		Addr     string
		Password string `json:",optional"`
		DB       int    `json:",default=0"`
		Prefix   string `json:",default=identity:authorization-version:"`
	}
	Token struct {
		Issuer                string `json:",default=hospital-identity"`
		Audience              string `json:",default=hospital-services"`
		AccessPublicKeyBase64 string
		AccessTTLSeconds      int64 `json:",default=900"`
	}
}
