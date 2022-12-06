package impl

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
)

const sendTimeout = 500 * time.Millisecond

// NewPeer creates a new peer. You can change the content and location of this
// function but you MUST NOT change its signature and package location.
func NewPeer(conf peer.Configuration) peer.Peer {
	// here you must return a struct that implements the peer.Peer functions.
	// Therefore, you are free to rename and change it as you want.
	// return &node{}
	RoutingTable := RoutingTable{}
	RoutingTable.Init(conf.Socket.GetAddress())

	node := &node{
		conf:         conf,
		stop:         0,
		RoutingTable: &RoutingTable,
		AddrMap: NewAddrMap(),
	}

	return node
}

// node implements a peer to build a Peerster system
//
// - implements peer.Peer
type node struct {
	peer.Peer

	// You probably want to keep the peer.Configuration on this struct:
	conf         peer.Configuration
	stop         int32
	WaitGroup    sync.WaitGroup
	RoutingTable *RoutingTable

	AddrMap      addressMap
}

type addressMap struct {
	mu sync.RWMutex
	addr map[string]string
}

func NewAddrMap() addressMap {
	return addressMap{
		addr: make(map[string]string),
	}
}

const defaultLevel = zerolog.NoLevel

var logout = zerolog.ConsoleWriter{
	Out:        os.Stdout,
	TimeFormat: time.RFC3339,
}

var Logger = zerolog.New(logout).Level(defaultLevel).
	With().Timestamp().Logger().
	With().Caller().Logger()

// Start implements peer.Service
func (n *node) Start() error {

	// 	The Start function starts listening on incoming messages with the socket. With the routing
	// table implemented in the next section, you must check if the message is for the node (i.e.
	// the packet’s destination equals the node’s socket address), or relay it if necessary. If the
	// packet is for the peer, then the registry must be used to execute the callback associated
	// with the message contained in the packet. If the packet is to be relayed, be sure to update
	// the RelayedBy field of the packet’s header to the peer’s socket address.

	n.conf.MessageRegistry.RegisterMessageCallback(types.ChatMessage{}, n.handleChatMessage)
	n.conf.MessageRegistry.RegisterMessageCallback(types.RegistrationMessage{}, n.handleRegistrationMessage)

	n.WaitGroup.Add(1)
	go n.Listen()

	return nil
	// panic("to be implemented in HW0")
}

// Stop implements peer.Service
func (n *node) Stop() error {
	//log.Info().Msg("stop the service")

	atomic.AddInt32(&n.stop, 1)

	//log.Debug().Msg("Waiting for all goroutines to stop")
	n.WaitGroup.Wait()
	//log.Debug().Msg("All go routines have stopped")

	return nil
	// panic("to be implemented in HW0")
}

// Unicast implements peer.Messaging
// check if there is a route to dest from routing table
// if it is, get the next forward address
func (n *node) Unicast(dest string, msg transport.Message) error {
	// panic("to be implemented in HW0")
	// log.Info().
	// 	Str("by", n.conf.Socket.GetAddress()).
	// 	Str("destination", dest).
	// 	Str("type", msg.Type).
	// 	Bytes("payload", msg.Payload).
	// 	Msg("send unicast message")

	if n.RoutingTable.IsRouteAvailable(dest) {
		forwardAddr, err := n.RoutingTable.GetForwardAddr(dest)
		if err != nil {
			log.Err(err).Msg("there is an error in getting next hop")
			return err
		}

		header := transport.NewHeader(n.conf.Socket.GetAddress(), n.conf.Socket.GetAddress(), dest, 0)
		packet := transport.Packet{
			Header: &header,
			Msg:    &msg,
		}
		return n.conf.Socket.Send(forwardAddr, packet, 500 * time.Millisecond)
	}

	return errors.New("no route available to dest")
}

// AddPeer implements peer.Service
func (n *node) AddPeer(addr ...string) {
	// panic("to be implemented in HW0")
	//log.Debug().Str("address", n.conf.Socket.GetAddress()).Msg("add peers")
	n.RoutingTable.AddPeers(addr)
}

// GetRoutingTable implements peer.Service
func (n *node) GetRoutingTable() peer.RoutingTable {
	// panic("to be implemented in HW0")
	// log.Debug().Msg("get routing table")

	return n.RoutingTable.GetRoutingTable()
}

// SetRoutingEntry implements peer.Service
func (n *node) SetRoutingEntry(origin, relayAddr string) {
	// panic("to be implemented in HW0")
	n.RoutingTable.SetRoutingEntry(origin, relayAddr)
}

// This listener should be started once, on a goroutine
func (n *node) Listen() {

	defer n.WaitGroup.Done()

	for {
		
		if atomic.LoadInt32(&n.stop) > 0 {
			return
		}

		pkt, err := n.conf.Socket.Recv(time.Millisecond * 40)
		if errors.Is(err, transport.TimeoutError(0)) {
			continue
		} else if err != nil {
			log.Err(err).Msg("socket recv has error")
		}

		/* no error, continue on processing the received data */

		// check if the message dest equals current node address
		// if it is, execute message callback
		// if not, relay the message and  RelayedBy field needs to be updated and send the message to dest
		var e error
		if pkt.Header.Destination == n.conf.Socket.GetAddress() {
			e = n.conf.MessageRegistry.ProcessPacket(pkt)
		} else {
			pkt.Header.RelayedBy = n.conf.Socket.GetAddress()
			e = n.conf.Socket.Send(pkt.Header.Destination, pkt, 500 * time.Millisecond)
		}
		if e != nil{
			log.Info().Err(e)
		}
	}
}

// When you receive a packet from the socket, you must call ProcessPacket with the packet received, which will call
// the callback registered with RegisterMessageCallback for that message
func (n *node) handleChatMessage(msg types.Message, pkt transport.Packet) error {

	_, exists := msg.(*types.ChatMessage)
	if !exists {
		return xerrors.Errorf("wrong type: %T", msg)
	}

	return nil
}

// When you receive a packet from the socket, you must call ProcessPacket with the packet received, which will call
// the callback registered with RegisterMessageCallback for that message
func (n *node) handleRegistrationMessage(msg types.Message, pkt transport.Packet) error {

	_, exists := msg.(*types.RegistrationMessage)
	if !exists {
		return xerrors.Errorf("wrong type: %T", msg)
	}
	fmt.Println("receive registration")
	n.AddrMap.mu.Lock()
	n.AddrMap.addr[pkt.Header.Source] = n.conf.Socket.GetAddress()
	n.AddrMap.mu.Unlock()

	fmt.Println(pkt.Header.Source)

	header := transport.NewHeader(n.conf.Socket.GetAddress(), n.conf.Socket.GetAddress(), pkt.Header.Source, 0)
	ackMsg := types.ChatMessage{
		Message: "registration success",
	}
	ackPkt := transport.Packet{
		Header: &header,
		Msg:    n.getMarshalledMsg(&ackMsg),
	}

	if err := n.conf.Socket.Send(pkt.Header.Source, ackPkt, sendTimeout); err != nil {
		log.Err(err).Msg("failed to send ack")
		return err
	}


	return nil
}


// Convert types.Message to *transport.Message.
// Returns nil if marshall failed.
func (n *node) getMarshalledMsg(msg types.Message) *transport.Message {
	m, err := n.conf.MessageRegistry.MarshalMessage(msg)
	if err != nil {
		log.Err(err).Msg("failed to marshall message")
		return nil
	}
	return &m
}
