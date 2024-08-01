package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/pion/turn/v3"
)

func main() {
	// TODO
	publicIP := "0.0.0.0" //! Local Test
	address := "0.0.0.0:3478"
	users := "user123"
	realm := "mfakhrullah.com"

	if len(publicIP) == 0 {
		log.Fatalf("'public-ip' is required")
	} else if len(users) == 0 {
		log.Fatalf("'users' is required")
	}

	// Create a UDP listener to pass into pion/turn
	// pion/turn itself doesn't allocate any UDP sockets, but lets the user pass them in
	// this allows us to add logging, storage or modify inbound/outbound traffic
	udpListener, err := net.ListenPacket("udp4", address)
	if err != nil {
		log.Panicf("Failed to create TURN server listener: %s", err)
	}

	fmt.Println("Listening to:", address)

	s, err := turn.NewServer(turn.ServerConfig{
		Realm: realm,
		// Set AuthHandler callback
		// This is called every time a user tries to authenticate with the TURN server
		// Return the key for that user, or false when no user is found
		AuthHandler: func(username string, realm string, srcAddr net.Addr) ([]byte, bool) { // nolint: revive
			log.Println("AuthHandler invoked for user:", username)
			if username == "user123" {
				return turn.GenerateAuthKey(username, realm, "pass123"), true
			}
			return nil, false
		},
		// PacketConnConfigs is a list of UDP Listeners and the configuration around them
		PacketConnConfigs: []turn.PacketConnConfig{
			{
				PacketConn: udpListener,
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: net.ParseIP(publicIP), // Claim that we are listening on IP passed by user (This should be your Public IP)
					Address:      "0.0.0.0",              // But actually be listening on every interface
				},
			},
		},
	})
	
	if err != nil {
		log.Panic(err)
	}

	// Block until user sends SIGINT or SIGTERM
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	if err = s.Close(); err != nil {
		log.Panic(err)
	}
}