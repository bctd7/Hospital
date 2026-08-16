package common

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// RequestFingerprint 为幂等请求生成紧凑的业务内容指纹。
// 各输入类型通过 json:"-" 排除 operation_id 和 request_id，避免把传输批次误当成业务内容。
func RequestFingerprint(payload any) string {
	data, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
