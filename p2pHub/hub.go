package main

import (
	"sync"

	"github.com/pion/webrtc/v4"
)

type UserConnection struct {
	peerConnection *webrtc.PeerConnection
	// Add other fields if necessary, like user ID, data channels, etc.
}

type Hub struct {
	connections map[string]*UserConnection
	mutex       sync.Mutex // To handle concurrent access to the connections map
}

func newHub() *Hub {
	return &Hub{
		connections: make(map[string]*UserConnection),
		// offer:   make(chan *Client),
		// answer: make(chan *Client),
	}
}

func (h *Hub) run() {
	
}