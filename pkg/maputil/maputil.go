/*
Package maputil provides a collection of map utility functions.
*/
package maputil

// Filter returns a new map containing
// only the elements in the input map m for which the specified function f is true.
func Filter[M ~map[K]V, K comparable, V any](m M, f func(K, V) bool) M {
	r := make(M, len(m))

	for k, v := range m {
		if f(k, v) {
			r[k] = v
		}
	}

	return r
}

// Map returns a new map that contains
// each of the elements of the input map m mutated by the specified function.
func Map[M ~map[K]V, K, J comparable, V, U any](m M, f func(K, V) (J, U)) map[J]U {
	r := make(map[J]U, len(m))

	for k, v := range m {
		j, u := f(k, v)
		r[j] = u
	}

	return r
}

// Reduce applies the reducing function f
// to each element of the input map m, and returns the value of the last call to f.
// The first parameter of the reducing function f is initialized with init.
func Reduce[M ~map[K]V, K comparable, V, U any](m M, init U, f func(K, V, U) U) U {
	r := init

	for k, v := range m {
		r = f(k, v, r)
	}

	return r
}

// Invert returns a new map were keys and values are swapped.
func Invert[M ~map[K]V, K, V comparable](m M) map[V]K {
	r := make(map[V]K, len(m))

	for k, v := range m {
		r[v] = k
	}

	return r
}
