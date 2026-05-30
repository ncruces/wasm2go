package passes

type set[T comparable] map[T]struct{}

func (s set[T]) add(t T) bool {
	if _, ok := s[t]; ok {
		return false
	}
	s[t] = struct{}{}
	return true
}

func (s set[T]) del(t T) bool {
	if _, ok := s[t]; ok {
		delete(s, t)
		return true
	}
	return false
}

func (s set[T]) has(t T) bool {
	_, ok := s[t]
	return ok
}
