package utils

func PtrToStringOrDefault(ptr *string, defaultVal string) string {
	if ptr == nil {
		return defaultVal
	}
	return *ptr
}

func StringToPtr(s string) *string {
	if s == "" {
		return nil // Save as NULL in database if empty
	}
	return &s
}
