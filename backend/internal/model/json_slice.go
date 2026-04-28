package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// StringSlice 支持数据库存储为 JSONB 的字符串切片
type StringSlice []string

// Value 实现 driver.Valuer 接口
func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// Scan 实现 sql.Scanner 接口
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, s)
	case string:
		return json.Unmarshal([]byte(v), s)
	default:
		return fmt.Errorf("无法将 %T 扫描到 StringSlice", value)
	}
}
