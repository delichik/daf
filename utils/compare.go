package utils

import "reflect"

func DeepCompare[T comparable](a T, b any) bool {
	tb, ok := b.(T)
	if !ok {
		panic("struct is not matched")
	}
	return reflect.DeepEqual(tb, a)
}
