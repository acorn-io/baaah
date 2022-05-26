package typed

func Debounce[T any](in <-chan T) <-chan T {
	result := make(chan T, 1)
	go func() {
		for msg := range in {
			select {
			case result <- msg:
			default:
			}
		}
	}()
	return result
}

func Tee[T any](in <-chan T) (<-chan T, <-chan T) {
	one := make(chan T, 1)
	two := make(chan T, 1)

	go func() {
		defer close(one)
		defer close(two)
		for x := range in {
			one <- x
			two <- x
		}
	}()

	return one, two
}
