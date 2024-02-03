package store

import (
	"github.com/DAO-Metaplayer/metachain/types"
)

// Utilities for test
const (
	TestEpochSize = 100
)

type MockBlockchain struct {
	HeaderFn            func() *types.Header
	GetHeaderByNumberFn func(uint64) (*types.Header, bool)
}

func (m *MockBlockchain) Header() *types.Header {
	return m.HeaderFn()
}

func (m *MockBlockchain) GetHeaderByNumber(height uint64) (*types.Header, bool) {
	return m.GetHeaderByNumberFn(height)
}
