package sungrow

type Models []string

func (m Models) Contains(v string) bool {
	for _, x := range m {
		if x == v {
			return true
		}
	}
	return false
}

func (m Models) ContainsOrNull(v string) bool {
	return len(m) == 0 || m.Contains(v)
}
