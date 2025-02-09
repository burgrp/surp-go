package surp

import "fmt"

type Optional[T any] struct {
	value   T
	defined bool
}

func NewOptional[T any](value T, defined bool) Optional[T] {
	return Optional[T]{
		value:   value,
		defined: defined,
	}
}

func NewDefined[T any](value T) Optional[T] {
	return Optional[T]{
		value:   value,
		defined: true,
	}
}

func NewUndefined[T any]() Optional[T] {
	return Optional[T]{
		defined: false,
	}
}

func (o Optional[T]) Get() T {
	if !o.defined {
		panic("Optional value is undefined")
	}
	return o.value
}

func (o Optional[T]) IsDefined() bool {
	return o.defined
}

func (o Optional[T]) IsUndefined() bool {
	return !o.defined
}

func (o Optional[T]) GetOrDefault(defaultValue T) T {
	if !o.defined {
		return defaultValue
	}
	return o.value
}

func (o Optional[T]) String() string {
	if !o.defined {
		return "(undefined)"
	}
	return fmt.Sprintf("%v", o.value)
}
