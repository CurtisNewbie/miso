package ptr

func BoolPtr(v bool) *bool {
	return &v
}

func StrPtr(v string) *string {
	return &v
}

func IntPtr(v int) *int {
	return &v
}
