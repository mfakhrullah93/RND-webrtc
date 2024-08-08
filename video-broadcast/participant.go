package main

import (
	"encoding/json"
	"log"

	"github.com/pion/webrtc/v4"
)

type Participant struct {
	peer *Peer
}

func (pt *Participant) handleActivity(client *Client, sdp string) bool {
	pt.initPeerConnection()
	pt.peer.handleIceCandidate(client)
	answer := pt.peer.createAnswer(sdp)

	json, err := json.Marshal(answer)
	if err != nil {
		log.Printf("Failed to Marshal message: %s\n", err.Error())
		return false
	}

	client.hub.sendAnswer <- json

	return true
}

func (pt *Participant) initPeerConnection() {

	// New Connection
	peerConnection, err := webrtc.NewPeerConnection(peerconfig)
	if err != nil {
		log.Printf("Failed to create new Peer Connection: %s\n", err.Error())
	}

	pt.peer.conn = peerConnection
	pt.peer.handleConnectionState()

	// TODO add track from localtrack published by publisher
	// rtpSender, err := peerConnection.AddTrack(localTrack)
	// if err != nil {
	// 	panic(err)
	// }
}