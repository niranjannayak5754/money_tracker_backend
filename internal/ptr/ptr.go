package ptr

func Bool(v bool) *bool {
	return &v
}

func String(v string) *string {
	return &v
}

func Int(v int) *int {
	return &v
}
