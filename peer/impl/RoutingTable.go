package impl

import (
	"errors"
	"sync"

	"go.dedis.ch/cs438/peer"
)


// The routing table is used to get the next-hop of a packet based on its destination. It should
// be used each time a peer receives a packet on its socket, and updated with the AddPeer
// and SetRoutingEntry functions.
// A routing table can be constructed by map, where key is the original node and the value is the relay address.
// A routing table always has an entry of itself
//	Table[myAddr] = myAddr.
//  Table[C] = B means that to reach C, message must be sent to B to relay.
type RoutingTable struct {
	route map[string]string
	sync.RWMutex
}

// Once add a peer, it needs to add itself to routing table
func (RT *RoutingTable) AddPeers(addresses []string) {
	for _, address := range addresses {
		RT.SetRoutingEntry(address, address)
	}
}

// GetRoutingTable returns the node's routing table. It should be a copy.
func (RT *RoutingTable) GetRoutingTable() peer.RoutingTable {
	RT.RLock()
	defer RT.RUnlock()

	RTCopy := make(peer.RoutingTable)
	for key, value := range RT.route {
		RTCopy[key] = value
	}

	return RTCopy
}


// SetRoutingEntry sets the routing entry. Overwrites it if the entry already exists. 
// If the origin is equal to the relayAddr, then the node has a new neighbor.
// If relayAddr is empty then the record must be deleted (and the peer has potentially lost a neighbor).
func (RT *RoutingTable) SetRoutingEntry(origin, relayAddr string) {
	RT.Lock()
	defer RT.Unlock()

	// log.Debug().Msgf("Add Peer %v %v", origin, relayAddr)
	// defer log.Debug().Msgf("After adding %v", RT.route)

	if relayAddr == "" {
		delete(RT.route, origin)
		return
	}

	if relay, exists := RT.route[origin]; exists {
		if relay == origin {
			return
		}
	}

	RT.route[origin] = relayAddr
}


// Initial the routing table and add itself
func (RT *RoutingTable) Init(origin string) {
	RT.route = make(map[string]string)

	RT.SetRoutingEntry(origin, origin)
}


// given dest, judge if the dest in the routing table
func (RT *RoutingTable) IsRouteAvailable(dest string) bool {
	RT.RLock()
	defer RT.RUnlock()

	if _, exists := RT.route[dest]; exists {
		return true
	}
	return false
}

// get the next forward address from routing table
func (RT *RoutingTable) GetForwardAddr(dest string) (string, error) {
	RT.RLock()
	defer RT.RUnlock()

	if ForwardAddr, exists := RT.route[dest]; exists {
		return ForwardAddr, nil
	}
	return "", errors.New("no next hop available in routing table")
}


