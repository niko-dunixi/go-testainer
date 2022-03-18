package common

func Identity[A any](item A) A {
	return item
}

type ConvertFunc[A, B any] func(a A) B
