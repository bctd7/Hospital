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
	WeChat struct {
		AppID           string `json:",optional"`
		AppSecret       string `json:",optional"`
		Code2SessionURL string `json:",default=https://api.weixin.qq.com/sns/jscode2session"`
	}
	PhoneLogin struct {
		Enabled         bool   `json:",default=false"`
		AccessKeyID     string `json:",optional"`
		AccessKeySecret string `json:",optional"`
		RegionID        string `json:",default=cn-shanghai"`
		Endpoint        string `json:",default=dypnsapi.aliyuncs.com"`
		SignName        string `json:",optional"`
		TemplateCode    string `json:",optional"`
		SchemeName      string `json:",optional"`
		ValidSeconds    int64  `json:",default=300"`
		IntervalSeconds int64  `json:",default=60"`
		CodeLength      int64  `json:",default=6"`
	}
	PhoneLookupKeyBase64 string
}
