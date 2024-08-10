package main

import (
	"encoding/json"
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

func (p *Peer)createAnswer(offer string) *webrtc.SessionDescription {
	// Set the remote SessionDescription
	err := p.conn.SetRemoteDescription(webrtc.SessionDescription{SDP: offer, Type: webrtc.SDPTypeOffer})
	if err != nil {
		log.Printf("[Peer] Connection Failed:: %s\n", err.Error())
	}

	// Create answer
	answer, err := p.conn.CreateAnswer(nil)
	if err != nil {
		log.Printf("[Peer] Connection Failed: %s", err.Error())
	}

	// Sets the LocalDescription, and starts our UDP listeners
	err = p.conn.SetLocalDescription(answer)
	if err != nil {
		panic(err)
	}
	
	return p.conn.LocalDescription()
}

func (p *Peer)addCandidate(candidate interface{}) {
	json, err := json.Marshal(candidate)
	if err != nil {
		log.Printf("[Peer] Internal Server Error: %s", err.Error())
		return
	}	

	if candidateErr := p.conn.AddICECandidate(webrtc.ICECandidateInit{Candidate: string(json)}); candidateErr != nil {
		log.Printf("[Peer] Connection Failed: %s", candidateErr.Error())
	}
}