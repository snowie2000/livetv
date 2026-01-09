package syncx

import (
	"fmt"
	"sync"
)

// Hashable is an interface that any type must implement to be stored in our data structure.
// The Digest() method should return a unique string key for the object.
type Hashable interface {
	Digest() string
}

// HashedSlice is a data structure that allows both indexed access and fast key-based search.
type HashedSlice[T Hashable] struct {
	mu     sync.RWMutex
	slice  []T
	lookup map[string]int
}

// NewFastSearchableSlice creates and returns an empty HashedSlice.
func NewHashedSlice[T Hashable]() *HashedSlice[T] {
	return &HashedSlice[T]{
		slice:  make([]T, 0),
		lookup: make(map[string]int),
	}
}

// NewFastSearchableSliceFromSlice converts a slice of Hashable items into a HashedSlice.
// It returns an error if any duplicate digests are found in the source slice.
func NewHashedSliceFromSlice[T Hashable](items []T) (*HashedSlice[T], error) {
	// Pre-allocate memory for efficiency
	lookup := make(map[string]int, len(items))
	// Create a copy of the slice to avoid modifying the original
	sliceCopy := make([]T, len(items))
	copy(sliceCopy, items)

	for i, item := range sliceCopy {
		digest := item.Digest()
		if _, exists := lookup[digest]; exists {
			return nil, fmt.Errorf("duplicate digest found in source slice: '%s'", digest)
		}
		lookup[digest] = i
	}

	return &HashedSlice[T]{
		slice:  sliceCopy,
		lookup: lookup,
	}, nil
}

func (fss *HashedSlice[T]) AsSlice() []T {
	fss.mu.RLock()
	defer fss.mu.RUnlock()
	return fss.slice
}

// Add appends a new item to the data structure.
// It returns an error if an item with the same digest already exists.
func (fss *HashedSlice[T]) Add(item T) error {
	fss.mu.Lock()
	defer fss.mu.Unlock()

	digest := item.Digest()
	if _, exists := fss.lookup[digest]; exists {
		return fmt.Errorf("item with digest '%s' already exists", digest)
	}

	fss.slice = append(fss.slice, item)
	fss.lookup[digest] = len(fss.slice) - 1
	return nil
}

func (fss *HashedSlice[T]) Each(f func(item T) bool) {
	fss.mu.RLock()
	defer fss.mu.RUnlock()
	for _, item := range fss.slice {
		if !f(item) {
			break
		}
	}
}

// GetByIndex returns an item by its index in the slice.
// It returns false if the index is out of bounds.
func (fss *HashedSlice[T]) GetByIndex(index int) (T, bool) {
	fss.mu.RLock()
	defer fss.mu.RUnlock()

	if index < 0 || index >= len(fss.slice) {
		var zero T
		return zero, false
	}
	return fss.slice[index], true
}

// GetByDigest searches for an item by its digest (key).
// It returns false if no item with that digest is found.
func (fss *HashedSlice[T]) GetByDigest(digest string) (T, bool) {
	fss.mu.RLock()
	defer fss.mu.RUnlock()

	index, found := fss.lookup[digest]
	if !found {
		var zero T
		return zero, false
	}
	return fss.slice[index], true
}

// Len returns the number of items in the data structure.
func (fss *HashedSlice[T]) Len() int {
	fss.mu.RLock()
	defer fss.mu.RUnlock()
	return len(fss.slice)
}
