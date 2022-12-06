package udp

import (
	"errors"
	"net"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.dedis.ch/cs438/transport"
)

const bufSize = 65000


// NewUDP returns a new udp transport implementation.
func NewUDP() transport.Transport {
	return &UDP{}
}

// UDP implements a transport layer using UDP
//
// - implements transport.Transport
type UDP struct {
}

// CreateSocket implements transport.Transport
func (n *UDP) CreateSocket(address string) (transport.ClosableSocket, error) {
	// panic("to be implemented in HW0")

	newUDPSocket, err := net.ListenPacket("udp", address)
	if err != nil {
		log.Err(err).Msg("failed to create a new UDP socket")
		return nil, err
	}

	return &Socket{
		socket:     newUDPSocket,
		messageIn:  make([]transport.Packet, 0),
		messageOut: make([]transport.Packet, 0),
	}, nil
}

// Socket implements a network socket using UDP.
//
// - implements transport.Socket
// - implements transport.ClosableSocket
type Socket struct {
	socket net.PacketConn

	messageIn  []transport.Packet
	messageOut []transport.Packet

	sync.Mutex
}

// Close implements transport.Socket. It returns an error if already closed.
func (s *Socket) Close() error {
	//log.Info().Str("address", s.GetAddress()).Msg("close socket")

	s.Lock()
	defer s.Unlock()

	return s.socket.Close()
	// panic("to be implemented in HW0")
}

// Send implements transport.Socket
func (s *Socket) Send(dest string, pkt transport.Packet, timeout time.Duration) error {
	// panic("to be implemented in HW0")
	s.Lock()
	defer s.Unlock()

	if timeout.Nanoseconds() != 0 {
		err := s.socket.SetWriteDeadline(time.Now().Add(timeout))
		if err != nil {
			log.Err(err).Msgf("There is an error in SetWriteDeadline")
			return err
		}
	} else {
		err := s.socket.SetWriteDeadline(time.Time{})
		if err != nil {
			log.Err(err).Msgf("There is an error in SetWriteDeadline")
			return err
		}
	}

	dst, err := net.ResolveUDPAddr("udp", dest)
	if err != nil {
		log.Err(err).Msgf("failed to resolve UDP address %s", dest)
		return err
	}

	pktBuffer, err := pkt.Marshal()
	if err != nil {
		log.Err(err).Msgf("failed to marshal packet data")
		return err
	}

	_, err = s.socket.WriteTo(pktBuffer, dst)

	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			err = transport.TimeoutError(timeout)
		}
		return err
	}

	pktCopy := pkt.Copy() // the pkt object might be shared
	s.messageOut = append(s.messageOut, pktCopy)

	return nil
}

// Recv implements transport.Socket. It blocks until a packet is received, or
// the timeout is reached. In the case the timeout is reached, return a
// TimeoutErr.
func (s *Socket) Recv(timeout time.Duration) (transport.Packet, error) {
	// panic("to be implemented in HW0")
	s.Lock()
	defer s.Unlock()

	packet := transport.Packet{}

	if timeout.Nanoseconds() != 0 {
		err := s.socket.SetReadDeadline(time.Now().Add(timeout))
		if err != nil {
			log.Err(err).Msgf("failed to SetReadDeadline")
			return packet, err
		}
	} else {
		err := s.socket.SetReadDeadline(time.Time{})
		if err != nil {
			log.Err(err).Msgf("failed to SetReadDeadline")
			return packet, err
		}
	}

	buffer := make([]byte, bufSize)
	n, addr, err := s.socket.ReadFrom(buffer)

	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			err = transport.TimeoutError(timeout)
		}
		return transport.Packet{}, err
	}

	buffer = buffer[:n]
	err = packet.Unmarshal(buffer)
	if err != nil {
		log.Err(err).Msgf("failed to unmarshal UDP packet from %s", addr)
		return packet, err
	}

	packetCopy := packet.Copy()
	s.messageIn = append(s.messageIn, packetCopy)

	return packet, nil
}

// GetAddress implements transport.Socket. It returns the address assigned. Can
// be useful in the case one provided a :0 address, which makes the system use a
// random free port.
func (s *Socket) GetAddress() string {
	s.Lock()
	defer s.Unlock()

	return s.socket.LocalAddr().String()
	// panic("to be implemented in HW0")
}

// GetIns implements transport.Socket
func (s *Socket) GetIns() []transport.Packet {
	s.Lock()
	defer s.Unlock()

	in := make([]transport.Packet, len(s.messageIn))

	for i, pkt := range s.messageIn {
		in[i] = pkt.Copy()
	}

	return in
	// panic("to be implemented in HW0")
}

// GetOuts implements transport.Socket
func (s *Socket) GetOuts() []transport.Packet {
	s.Lock()
	defer s.Unlock()

	out := make([]transport.Packet, len(s.messageOut))

	for i, packet := range s.messageOut {
		out[i] = packet.Copy()
	}

	return out
	// 	panic("to be implemented in HW0")
}
