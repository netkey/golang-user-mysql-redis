package utils

import (
	"errors"
	"github.com/speps/go-hashids/v2"
)

var (
	hd *hashids.HashID
	// ErrInvalidID 定义无效 ID 错误
	ErrInvalidID = errors.New("invalid id format")
)

// InitHashID 初始化，建议在 main.go 启动时或 init 中调用
func InitHashID(salt string, minLength int) {
	d := hashids.NewData()
	d.Salt = salt
	d.MinLength = minLength
	var err error
	hd, err = hashids.NewWithData(d)
	if err != nil {
		panic("HashID 初始化失败: " + err.Error())
	}
}

// EncodeID 将数字 ID 加密为字符串
func EncodeID(id int) (string, error) {
	return hd.Encode([]int{id})
}

// DecodeID 将字符串解密为数字 ID
func DecodeID(hash string) (int, error) {
	ids, err := hd.DecodeWithError(hash)
	if err != nil || len(ids) == 0 {
		return 0, ErrInvalidID
	}
	return ids[0], nil
}
