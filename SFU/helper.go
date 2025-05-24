package sfu

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefUint(i *uint16) uint16 {
	if i == nil {
		return 0
	}
	return *i
}