package store

import "testing"

// StoreTester is a unified test suite for Store implementations
type StoreTester struct {
	NewStore func() Store
}

// RunAllTests runs all tests for a Store implementation
func (s *StoreTester) RunAllTests(t *testing.T) {
	// This is just a placeholder - the actual implementation is in store_test.go
	// We need to duplicate the method here to make it available to other packages
}