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
