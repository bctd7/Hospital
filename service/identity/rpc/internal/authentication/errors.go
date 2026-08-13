// Package authentication 定义手机号凭据认证所需的领域端口和错误。
// 它只负责确认登录者控制某个手机号，不管理账号资料，也不直接签发 Token。
package authentication

import "errors"

var ErrInvalidPhone = errors.New("invalid phone number")
