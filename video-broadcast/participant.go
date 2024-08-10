package main

import (
	"encoding/json"
	"log"

	"github.com/pion/webrtc/v4"
)

type Participant struct {
	hub *Hub
	peer *Peer

	// Participant stream is set from publisher stream
	stream *Stream

	receivedtrack chan bool
}

func (pt *Participant) handlePeerConnection(sdp string) bool {
	pt.initPeerConnection()

	pt.stream.join <- pt

	ok := <-pt.receivedtrack
	if !ok { return false }

	pt.handleIceCandidate()
	answer := pt.peer.createAnswer(sdp)

	msg := Message{P2P: P2P{Activity: JOIN, Answer: Offer{SDP: answer.SDP, Type: answer.Type.String()}}}
	
	json, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to Marshal message: %s\n", err.Error())
		return false
	}	

	pt.hub.sendAnswer <- json

	return true
}

func (pt *Participant) initPeerConnection() {

	// New Connection
	peerConnection, err := webrtc.NewPeerConnection(peerconfig)
	if err != nil {
		log.Printf("Failed to create new Peer Connection: %s\n", err.Error())
	}

	pt.peer.conn = peerConnection
	pt.handleConnectionState()
}

func (pt *Participant) handleConnectionState() {
	// Set the handler for Peer connection state
	// This will notify you when the peer has connected/disconnected
	pt.peer.conn.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		log.Printf("Participant Connection State has changed: %s\n", s.String())

		if s == webrtc.PeerConnectionStateFailed {
			// Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
			// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
			// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
			log.Println("Participant Connection has gone to failed exiting")
		}

		if s == webrtc.PeerConnectionStateClosed {
			// PeerConnection was explicitly closed. This usually happens from a DTLS CloseNotify
			log.Println("Participant Connection has gone to closed exiting")		
		}

	})	
}

func (pt *Participant)handleIceCandidate() {	
	
	msg := Message{P2P: P2P{Activity: JOIN}}

	// When an ICE candidate is available send to the user Peer
	pt.peer.conn.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil { return }

		msg.P2P.Candidate = c
		
		json, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Failed to Marshal message: %s\n", err.Error())
		}	
		
		pt.hub.sendCandidate <- json
	})
}