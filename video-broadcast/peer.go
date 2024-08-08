package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/pion/webrtc/v4"
)

var peerconfig = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			URLs: []string{"stun:stun.l.google.com:19302"},
		},
	},
}

type Peer struct {
	conn 	*webrtc.PeerConnection
}

func (p *Peer)handleConnectionState() {
	// Set the handler for Peer connection state
	// This will notify you when the peer has connected/disconnected
	p.conn.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		log.Printf("Peer Connection State has changed: %s\n", s.String())

		if s == webrtc.PeerConnectionStateFailed {
			// Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
			// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
			// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
			log.Println("Peer Connection has gone to failed exiting")
		}

		if s == webrtc.PeerConnectionStateClosed {
			// PeerConnection was explicitly closed. This usually happens from a DTLS CloseNotify
			log.Println("Peer Connection has gone to closed exiting")		
		}

	})	
}

func (p *Peer)handleIceCandidate(client *Client) {	

	// When an ICE candidate is available send to the user Peer
	p.conn.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		fmt.Printf("candidate: %v", c)
		json, err := json.Marshal(c)
		if err != nil {
			log.Printf("Failed to Marshal message: %s\n", err.Error())
		}	
		
		client.hub.sendCandidate <- json
	})
}

func (p *Peer)createAnswer(offer string) *webrtc.SessionDescription {
	// Set the remote SessionDescription
	err := p.conn.SetRemoteDescription(webrtc.SessionDescription{SDP: offer, Type: webrtc.SDPTypeOffer})
	if err != nil {
		log.Printf("[createAnswer] Failed to SetRemoteDescription: %s\n", err.Error())
	}

	// Create answer
	answer, err := p.conn.CreateAnswer(nil)
	if err != nil {
		log.Printf("Failed to CreateAnswer: %s\n", err.Error())
	}

	// Sets the LocalDescription, and starts our UDP listeners
	err = p.conn.SetLocalDescription(answer)
	if err != nil {
		panic(err)
	}
	
	return p.conn.LocalDescription()
}

func (p *Peer)addCandidate(candidate interface{}) bool{
	json, err := json.Marshal(candidate)
	if err != nil {
		log.Printf("Failed to Marshal message: %s\n", err.Error())
		return false
	}	

	if candidateErr := p.conn.AddICECandidate(webrtc.ICECandidateInit{Candidate: string(json)}); candidateErr != nil {
		log.Printf("Failed to Add Candidate: %v", candidateErr.Error())
		return false
	}

	return true
}