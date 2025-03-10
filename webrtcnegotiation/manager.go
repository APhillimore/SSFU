package webrtcnegotiation

import (
	"errors"
	"slices"
	"sync"
)

type negotiatorList[T IWebRTCNegotiator] struct {
	negotiators []T
	mu          sync.RWMutex
}

func (hl *negotiatorList[T]) Add(negotiator T) T {
	hl.mu.Lock()
	defer hl.mu.Unlock()
	hl.negotiators = append(hl.negotiators, negotiator)
	return negotiator
}

func (hl *negotiatorList[T]) Remove(negotiator T) bool {
	hl.mu.Lock()
	defer hl.mu.Unlock()
	originalLen := len(hl.negotiators)
	hl.negotiators = slices.DeleteFunc(hl.negotiators, func(n T) bool {
		return n.ID() == negotiator.ID()
	})
	return len(hl.negotiators) < originalLen
}

func (hl *negotiatorList[T]) GetByID(id string) (T, error) {
	hl.mu.RLock()
	defer hl.mu.RUnlock()
	for _, n := range hl.negotiators {
		if n.ID() == id {
			return n, nil
		}
	}
	var zero T
	return zero, errors.New("negotiator not found")
}

func (hl *negotiatorList[T]) Count() int {
	hl.mu.RLock()
	defer hl.mu.RUnlock()
	return len(hl.negotiators)
}

type WebRTCNegotiationManager struct {
	Negotiators negotiatorList[*WebRtcNegotiator]
}

func NewWebRTCNegotiationManager() *WebRTCNegotiationManager {
	return &WebRTCNegotiationManager{
		Negotiators: negotiatorList[*WebRtcNegotiator]{},
	}
}
