package cache

import "errors"

var (
	// ErrInvalidDst 目标变量无效错误
	ErrInvalidDst = errors.New("invalid destination: must be a pointer")

	// ErrInvalidDstMap 目标map无效错误
	ErrInvalidDstMap = errors.New("invalid destination map: must be a pointer to map")

	// ErrTypeMismatch 类型不匹配错误
	ErrTypeMismatch = errors.New("type mismatch between source and destination")
)
