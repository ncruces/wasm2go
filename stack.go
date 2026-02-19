package main

func pop[S ~[]E, E any](s *S) E {
	i := len(*s) - 1
	e := (*s)[i]
	*s = (*s)[:i]
	return e
}
