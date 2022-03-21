package common

// Functional call, take an item and return the item.
// An implementation of the MapFunc[A, B any] where
// A and B are the same and the element returned is
// the same item passed in.
func Identity[A any](item A) A {
	return item
}

// Functional method for any object that is input and
// converted to another
type MapFunc[A, B any] func(a A) B
