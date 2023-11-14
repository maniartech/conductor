package choreo

type ActivityHandler[T any] func(T) bool

type ErrorHandler[T any] func(string, T, error)
