package main

type stack[E any] []E

func (s *stack[E]) top() *E {
	a := *s
	return &a[len(a)-1]
}

func (s *stack[E]) last(n int) []E {
	a := *s
	return a[len(a)-n:]
}

func (s *stack[E]) pop() E {
	a := *s
	i := len(a) - 1
	e := a[i]
	*s = a[:i]
	return e
}

func (s *stack[E]) append(e E) {
	a := *s
	*s = append(a, e)
}
