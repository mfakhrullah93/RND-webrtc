// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 5120

	PUBLISH = 1

	JOIN = 2
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		if(r.Header.Get("Origin") == "http://localhost:4200"){
			return true
		}
		if(r.Header.Get("Origin") == "http://localhost:8080"){
			return true
		}
		return false 
	},
}

type Offer struct {
	SDP   string `json:"sdp"`
	Type   string `json:"type"`
}

type P2P struct {
	Initiate 	Offer 			`json:"initiate"`
	Answer 	Offer 				`json:"answer"`
	Candidate interface{} `json:"candidate"`
	Activity 	int 				`json:"activity"`
}

type Message struct {
	Streamid StreamId	`json:"streamid"`
	P2P P2P `json:"p2p"`
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub 		*Hub

	// The websocket connection.
	wsConn 	*websocket.Conn

	// Client peer connection as publisher
	publisher *Publisher

	// Client peer connection as participant
	participant *Participant

	// Buffered channel of outbound messages.
	send 		chan []byte
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.wsConn.Close()
	}()
	c.wsConn.SetReadLimit(maxMessageSize)
	c.wsConn.SetReadDeadline(time.Now().Add(pongWait))
	c.wsConn.SetPongHandler(func(string) error { c.wsConn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.wsConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		// c.hub.broadcast <- message
		// log.Printf("JSON string Message: %s", message)
		
		var msg Message 
		err = json.Unmarshal(message, &msg)
		if err != nil {
			log.Fatalf("Error unmarshalling JSON: %v", err)
		}

		// log.Printf("received msg: %+v", msg)

		// Handle incoming activity as publisher
		if msg.P2P.Activity == PUBLISH { 			
			offer := &msg.P2P.Initiate

			if msg.P2P.Candidate != nil { 
				c.publisher.peer.addCandidate(msg.P2P.Candidate) 
			} else {				
				ok := c.publishStream(offer, msg.Streamid)
				if !ok { break; } // TODO dont break. notify error properly
			}
		}

		if msg.P2P.Activity == JOIN { 			
			offer := &msg.P2P.Initiate
			
			if msg.P2P.Candidate != nil { 
				c.participant.peer.addCandidate(msg.P2P.Candidate) 
			} else {
				ok := c.joinStream(offer, msg.Streamid)
				if !ok { break; } // TODO dont break. notify error properly
			}
		}

	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.wsConn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.wsConn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.wsConn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.wsConn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.wsConn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.wsConn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) publishStream(offer *Offer, id StreamId) bool {
	if offer.Type != "offer" { return false }
	
	newStream := newStream(id)
	c.publisher.stream = newStream
	c.hub.stream[id] = newStream

	return c.publisher.handlePeerConnection(offer.SDP)
}

func (c *Client) joinStream(offer *Offer, id StreamId) bool {
	if offer.Type != "offer" { return false }
	
	joinStream := c.hub.stream[id]
	c.participant.stream = joinStream

	return c.participant.handlePeerConnection(offer.SDP)
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	// Initialize peer.
	// Peer connection will be handle at incoming client message.
	// Peer connection is for publisher and participant.
	peer := &Peer{}
	publisher := &Publisher{ peer: peer, hub: hub }
	participant := &Participant{ peer: peer, hub: hub, receivedtrack: make(chan bool)}

	// New Client
	// There will be only 1 active peer connection, either as publisher or participant
	// decided at client incoming message
	client := &Client{
		publisher: publisher, 
		participant: participant, 
		hub: hub, 
		wsConn: wsConn, 
		send: make(chan []byte, 256),	
	}

	client.hub.register <- client
	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}