package utils

import (
	"strconv"
	"strings"
)

func ToInt(v any) int {
	switch val := v.(type) {
	case int:
		return val
	case int8:
		return int(val)
	case int16:
		return int(val)
	case int32:
		return int(val)
	case int64:
		return int(val)
	case uint:
		return int(val)
	case uint8:
		return int(val)
	case uint16:
		return int(val)
	case uint32:
		return int(val)
	case uint64:
		return int(val)
	case float32:
		return int(val)
	case float64:
		return int(val)
	case string:
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
		return 0
	default:
		return 0
	}
}

func ToString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case int, int8, int16, int32, int64:
		return strconv.FormatInt(int64(ToInt(val)), 10)
	case uint, uint8, uint16, uint32, uint64:
		return strconv.FormatUint(uint64(ToInt(val)), 10)
	case float32, float64:
		return strconv.FormatFloat(ToFloat64(val), 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	default:
		return ""
	}
}

func ToFloat64(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}

func Contains(s []string, substr string) bool {
	for _, str := range s {
		if strings.Contains(strings.ToLower(str), strings.ToLower(substr)) {
			return true
		}
	}
	return false
}

func Deduplicate(s []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, str := range s {
		if !seen[str] {
			seen[str] = true
			result = append(result, str)
		}
	}
	return result
}

func Ptr[T any](v T) *T {
	return &v
}
