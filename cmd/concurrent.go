package cmd

import "sync"

// runConcurrent executes fn for each item using up to concurrency goroutines.
// It returns all errors collected across workers.
func runConcurrent[T any](items []T, concurrency int, fn func(T) error) []error {
	ch := make(chan T, len(items))
	for _, item := range items {
		ch <- item
	}
	close(ch)

	workers := concurrency
	if workers > len(items) {
		workers = len(items)
	}
	if workers < 1 {
		workers = 1
	}

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range ch {
				if err := fn(item); err != nil {
					mu.Lock()
					errs = append(errs, err)
					mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()
	return errs
}
