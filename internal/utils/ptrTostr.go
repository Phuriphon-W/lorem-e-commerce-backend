package utils

func PtrToStringOrDefault(ptr *string, defaultVal string) string {
	if ptr == nil {
		return defaultVal
	}
	return *ptr
}
