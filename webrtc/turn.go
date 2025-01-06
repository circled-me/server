package webrtc

import (
	"fmt"
	"net"
	"strconv"

	"log"

	"github.com/pion/turn/v2"
)

const (
	realm = "circled.me"
)

type TurnServer struct {
	Port           int
	PublicIP       string
	TrafficMinPort int
	TrafficMaxPort int
	AuthFunc       func(userToken string) bool
	server         *turn.Server
}

func (ts *TurnServer) Start() (err error) {
	udpListener, err := net.ListenPacket("udp4", "0.0.0.0:"+strconv.Itoa(ts.Port))
	if err != nil {
		return fmt.Errorf("Failed to create TURN server listener: %s", err)
	}

	ts.server, err = turn.NewServer(turn.ServerConfig{
		Realm: realm,
		AuthHandler: func(username string, realm string, srcAddr net.Addr) ([]byte, bool) { // nolint: revive
			if ts.AuthFunc(username) {
				return turn.GenerateAuthKey(username, realm, username), true
			}
			return nil, false
		},
		// PacketConnConfigs is a list of UDP Listeners and the configuration around them
		PacketConnConfigs: []turn.PacketConnConfig{
			{
				PacketConn: udpListener,
				RelayAddressGenerator: &turn.RelayAddressGeneratorPortRange{
					RelayAddress: net.ParseIP(ts.PublicIP), // Claim that we are listening on IP passed by user (This should be your Public IP)
					Address:      "0.0.0.0",                // But actually be listening on every interface
					MinPort:      uint16(ts.TrafficMinPort),
					MaxPort:      uint16(ts.TrafficMaxPort),
				},
			},
		},
	})
	return
}

func (ts *TurnServer) Stop() {
	if err := ts.server.Close(); err != nil {
		log.Printf("Closing TURN server error: %v", err)
	}
}
