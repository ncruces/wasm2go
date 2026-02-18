package main

type set[T comparable] map[T]struct{}

func (s set[T]) add(t T) {
	s[t] = struct{}{}
}
