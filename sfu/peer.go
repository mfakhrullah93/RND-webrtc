package main

import (
	"log"
	"sync"

	"github.com/pion/webrtc/v4"
)


type Peer struct {
	conn  						*webrtc.PeerConnection

	pendingCandidates	[]*webrtc.ICECandidate

	// Registered channels.
	channels 					map[*Channel]bool

	// Register requests from the channels.
	register 					chan *Channel

	// Unregister requests from channels.
	unregister 				chan *Channel

	offer							chan *webrtc.SessionDescription

	icecandidate			chan *webrtc.ICECandidate
}

var peerconfig = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			URLs: []string{"stun:stun.l.google.com:19302"},
		},
	},
}

func newPeer() *Peer{
	// Create a new WebRTC API instance
	api := webrtc.NewAPI(webrtc.WithSettingEngine(webrtc.SettingEngine{}))
	
	// Create a new RTCPeerConnection
	peerConnection, err := api.NewPeerConnection(peerconfig)
	if err != nil {
		log.Printf("Failed to create new Peer Connection: %s\n", err.Error())
	}

	peer := &Peer{
		conn: 				peerConnection,
		pendingCandidates: 	make([]*webrtc.ICECandidate, 0),
		channels: 					make(map[*Channel]bool),
		offer: 							make(chan *webrtc.SessionDescription),
		icecandidate: 			make(chan *webrtc.ICECandidate),
		register:   				make(chan *Channel),
		unregister: 				make(chan *Channel),
	}

	peer.handleConnectionState()
	peer.handleIceCandidate()

	return peer 	
}

func (p *Peer)createOffer() *webrtc.SessionDescription {	

	/// Define the audio codec capabilities
	audioCodec := webrtc.RTPCodecCapability{
		MimeType: "audio/opus",
	}

	// Create an audio track
	audioTrack, err := webrtc.NewTrackLocalStaticRTP(audioCodec, "audio", "stream")
	if err != nil {
		log.Fatalf("Failed to create audio track: %v", err)
	}

	// Add the audio track to the peer connection
	_, err = p.conn.AddTrack(audioTrack)
	if err != nil {
		log.Fatalf("Failed to add track: %v", err)
	}

	// Create an offer to send to the other process
	offer, err := p.conn.CreateOffer(nil)
	if err != nil {
		log.Printf("Failed to Create an offer: %s\n", err.Error())
		return nil
	}
	// Sets the LocalDescription, and starts our UDP listeners
	// Note: this will start the gathering of ICE candidates
	if err := p.conn.SetLocalDescription(offer); err != nil {
		log.Printf("Failed to Set the LocalDescription: %s\n", err.Error())
		return nil
	}

	return &offer
}

// func (p *Peer)signalMessage() bool{
	
	// fmt.Println("miaw1")
	// // message := Message{P2P: P2P{Initiate: offer}}
	// msgToBroadcast, err := json.Marshal(message)
	// if err != nil {
	// 	log.Printf("Failed to Marshal message: %s\n", err.Error())
	// 	return false
	// }	
	
	// fmt.Println(message)
	// Send out our offer
	// channel.hub.broadcast <- BroadcastMessage{message: []byte("msgToBroadcast"), sender: p.signal}

// 	return true
// }

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

func (p *Peer)handleIceCandidate() {	
	var candidatesMux sync.Mutex
	
	// When an ICE candidate is available send to the user Peer
	p.conn.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}

		candidatesMux.Lock()
		defer candidatesMux.Unlock()
		
		desc := p.conn.RemoteDescription()
		
		// Here you would send the ICE candidate to the user peer
		if desc == nil {
			p.pendingCandidates = append(p.pendingCandidates, c)
		} else {
			p.icecandidate <- c
		}
	})
}

// func (p *Peer) close() {
// 	if cErr := p.conn.Close(); cErr != nil {
// 		log.Printf("cannot close peerConnection: %v\n", cErr)
// 	}
// }

func (p *Peer)run() {
	for {
		select {
		case channel := <-p.register:
			p.channels[channel] = true
		case offer := <-p.offer:
			log.Println(offer)
			// for channel := range p.channels { // TODO Loop Send to all clients
			// 	select {
			// 	case channel.send <- offer:
			// 	default:
			// 		close(client.send)
			// 		delete(h.clients, client)
			// 	}
			// }
		case candidate := <-p.icecandidate:
			log.Println(candidate)
		case channel := <-p.unregister:
			if _, ok := p.channels[channel]; ok {
				delete(p.channels, channel)
				close(channel.send)
			}
		}
	}
}