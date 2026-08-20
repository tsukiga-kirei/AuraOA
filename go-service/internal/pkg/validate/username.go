package validate

import "regexp"

// 登录用户名：1–100 位字母、数字、下划线，允许纯数字（对接 OA loginid）。
var loginUsernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]{1,100}$`)

// LoginUsernameRule 是用户名格式的中文说明，供校验失败时返回。
const LoginUsernameRule = "用户名只能包含英文字母、数字和下划线，长度 1–100 位"

// IsLoginUsername 校验 AuraOA 登录用户名（租户管理员、组织成员）。
func IsLoginUsername(username string) bool {
	return loginUsernameRe.MatchString(username)
}
