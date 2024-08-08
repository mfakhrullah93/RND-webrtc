package main

import (
	"fmt"

	"github.com/pion/webrtc/v4"
)

type ChannelId int
type Channel struct{
	id ChannelId
	localTrack chan *webrtc.TrackLocalStaticRTP
}

func (ch *Channel) openChannel() {
	for {
		recvOnlyOffer := webrtc.SessionDescription{}

		fmt.Println(recvOnlyOffer)
	}
}