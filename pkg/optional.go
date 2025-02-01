package surp

import "fmt"

type Optional[T any] struct {
	value T
	valid bool
}

func NewOptional[T any](value T, valid bool) Optional[T] {
	return Optional[T]{
		value: value,
		valid: valid,
	}
}

func NewValid[T any](value T) Optional[T] {
	return Optional[T]{
		value: value,
		valid: true,
	}
}

func NewInvalid[T any]() Optional[T] {
	return Optional[T]{
		valid: false,
	}
}

func (o Optional[T]) Get() T {
	if !o.valid {
		panic("Optional value is not valid")
	}
	return o.value
}

func (o Optional[T]) IsValid() bool {
	return o.valid
}

func (o Optional[T]) GetOrDefault(defaultValue T) T {
	if !o.valid {
		return defaultValue
	}
	return o.value
}

func (o Optional[T]) String() string {
	if !o.valid {
		return "(invalid)"
	}
	return fmt.Sprintf("%v", o.value)
}
