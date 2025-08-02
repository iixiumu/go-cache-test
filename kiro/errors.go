package cache

import "errors"

var (
	// ErrInvalidDestination 无效的目标变量
	ErrInvalidDestination = errors.New("invalid destination: must be a settable pointer")
	
	// ErrTypeMismatch 类型不匹配
	ErrTypeMismatch = errors.New("type mismatch: cannot assign source to destination")
	
	// ErrInvalidMapType 无效的map类型
	ErrInvalidMapType = errors.New("invalid map type: destination must be a pointer to map")
)