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
