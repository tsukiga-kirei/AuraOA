// Package hash 提供密码哈希与校验工具，基于 bcrypt 算法实现。
package hash

import "golang.org/x/crypto/bcrypt"

// bcryptCost 哈希计算强度，值越大越安全但耗时越长，12 是生产环境推荐值。
const bcryptCost = 12

// HashPassword 对明文密码进行 bcrypt 哈希，返回哈希字符串。
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword 校验明文密码与 bcrypt 哈希是否匹配，匹配返回 true。
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
