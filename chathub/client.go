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

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

// Offer is a SDP offer from webrtc peer connection.
type Offer interface{}

// P2P structure either 3 will be broadcast for p2p comm.
type P2P struct {
	Initiate Offer `json:"initiate"`
	Answer Offer `json:"answer"`
	Candidate interface{} `json:"candidate"`
}

// Message structure. 
// The Message string is to broadcast normal string message
// P2P is a struct for p2p comm
type Message struct {
	Message string `json:"message"`
	P2P P2P `json:"p2p"`
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		log.Printf("JSON string Message: %s", message)
		
		// Create an instance of the message struct
		var msgStruct Message

		// Unmarshal the JSON string into the struct
		err = json.Unmarshal(message, &msgStruct)
		if err != nil {
			log.Fatalf("Error unmarshalling JSON: %v", err)
		}

		// Log the unmarshalled struct in a readable format
		log.Printf("Unmarshalled Message: %+v", msgStruct)

		c.handlePumpToHub(&msgStruct)

		// c.hub.broadcast <- BroadcastMessage{message: message, sender: c}
	}
}

// Handle messages from websocket connection to pump to the hub 
func (c *Client) handlePumpToHub(message *Message) {
	var messageToPump []byte
	var err error

	// broadcast normal message
	if message.Message != "" { 
		messageToPump = []byte(message.Message)
		c.hub.broadcast <- BroadcastMessage{message: messageToPump, sender: c}

	} else { 
		// broadcast p2p message
		messageToPump, err = json.Marshal(message.P2P)
		if err != nil {
			log.Fatalf("Error marshalling JSON: %v", err)
		}
		c.hub.broadcast <- BroadcastMessage{message: messageToPump, sender: c}
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
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
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
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256)} // TODO New Client
	client.hub.register <- client																					// TODO Register Client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}