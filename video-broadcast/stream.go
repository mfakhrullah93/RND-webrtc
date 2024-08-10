package main

import (
	"fmt"

	"github.com/pion/webrtc/v4"
)

// Stream Id is either user id, or channel id
type StreamId int 
type Stream struct{
	id StreamId

	// The Stream track
	streamTrack chan *webrtc.TrackLocalStaticRTP

	// New offer to participate in the stream
	join chan *Participant
}

func newStream(id StreamId) *Stream{
	return &Stream{
		id: id,
		streamTrack: make(chan *webrtc.TrackLocalStaticRTP),
		join: make(chan *Participant),
	}
}

func (s *Stream) serveStream() {

	track := <- s.streamTrack
	fmt.Println("streaming is served", track)
	for {

		participant := <- s.join
		fmt.Println("new participant")
		fmt.Println("Add Track")

		rtpSender, err := participant.peer.conn.AddTrack(track)
		if err != nil {
			fmt.Println("Failed to add track")
			participant.receivedtrack <- false
		}

		// Read incoming RTCP packets
		// Before these packets are returned they are processed by interceptors. For things
		// like NACK this needs to be called.
		go func() {
			rtcpBuf := make([]byte, 1500)
			for {
				if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
					return
				}
			}
		}()

		participant.receivedtrack <- true
		
	}
}