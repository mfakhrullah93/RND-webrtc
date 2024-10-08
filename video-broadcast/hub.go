package main

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Active stream.
	stream map[StreamId]*Stream

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client	

	// send peer answer signal 
	sendAnswer chan []byte

	// send peer candidate signal 
	sendCandidate chan []byte
}

func newHub() *Hub {
	return &Hub{
		clients:    		make(map[*Client]bool),
		stream:    			make(map[StreamId]*Stream),
		broadcast:  		make(chan []byte),
		register:   		make(chan *Client),
		unregister: 		make(chan *Client),
		sendAnswer: 		make(chan []byte),
		sendCandidate: 	make(chan []byte),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case answer := <-h.sendAnswer:
			for client := range h.clients { // TODO Loop Send to all clients
				select {
				case client.send <- answer:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		case candidate := <-h.sendCandidate:
			for client := range h.clients { // TODO Loop Send to all clients
				select {
				case client.send <- candidate:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}