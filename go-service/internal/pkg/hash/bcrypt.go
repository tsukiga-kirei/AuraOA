package hash

import "golang.org/x/crypto/bcrypt"

const bcryptCost = 12

//HashPassword 使用 cost=12 返回给定密码的 bcrypt 哈希值。
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

//CheckPassword 将明文密码与 bcrypt 哈希进行比较。
//如果匹配则返回 true。
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
