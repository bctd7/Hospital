package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	Environment string `json:",default=local"`
	MySQL       struct {
		DataSource string
	}
	SessionRedis struct {
		Addr                       string
		Password                   string `json:",optional"`
		DB                         int    `json:",default=0"`
		Prefix                     string `json:",default=identity:refresh:"`
		AuthorizationVersionPrefix string `json:",default=identity:authorization-version:"`
	}
	Kafka struct {
		Enabled                           bool   `json:",default=false"`
		Brokers                           string `json:",optional"`
		ClientID                          string `json:",default=identity-rpc"`
		AuthorizationChangedTopic         string `json:",default=identity.authorization.changed.v1"`
		AuthorizationVersionConsumerGroup string `json:",default=identity-authorization-version-consumer-v1"`
		BatchSize                         int    `json:",default=100"`
	}
	Token struct {
		Issuer                 string `json:",default=hospital-identity"`
		Audience               string `json:",default=hospital-services"`
		AccessPrivateKeyBase64 string
		AccessPublicKeyBase64  string
		AccessTTLSeconds       int64 `json:",default=900"`
		RefreshTTLSeconds      int64 `json:",default=2592000"`
	}
	PhoneLogin struct {
		Verifier        string `json:",optional"`
		LocalCode       string `json:",optional"`
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
