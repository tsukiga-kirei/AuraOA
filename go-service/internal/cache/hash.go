// Package cache 提供统一的缓存管理能力
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// ComputeFilterHash 计算筛选条件的哈希值
// 接受任意可序列化的参数，返回 16 字符的十六进制字符串（SHA256 前 8 字节）
// 用于生成缓存键中的 filter_hash 部分，确保相同筛选条件生成相同的缓存键
func ComputeFilterHash(params interface{}) string {
	data, err := json.Marshal(params)
	if err != nil {
		// 序列化失败时返回空字符串，调用方应处理此情况
		return ""
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:8]) // 取前 8 字节，生成 16 字符的十六进制字符串
}
