package main

import (
	"flag"
	"log"
	"net/http"
)

var addr = flag.String("addr", ":8080", "http service address")

func main() {
	flag.Parse()

	peer := newPeer()
	go peer.run()
	
	http.HandleFunc("/channel", func(w http.ResponseWriter, r *http.Request){		
		serveChannel(peer, w, r)
	})
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}	

	// peerSignal := PeerSignal{wsConn: wsConn, done: make(chan bool), send: make(chan []byte, 256)}
	
	// interrupt := make(chan os.Signal, 1)
	// signal.Notify(interrupt, os.Interrupt)

	// go peerSignal.run()	

	// for {
	// 	select{
	// 	case <-interrupt:
	// 		log.Println("Interrupt")
	// 		err := peerSignal.wsConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	// 		if err != nil {
	// 			log.Println("write close:", err)
	// 			return
	// 		}
	// 	case <- peerSignal.done:
	// 		log.Println("Shutting down")
	// 		return
	// 	}	
	// }
	
}