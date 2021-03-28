package peers

import (
	"encoding/binary"
	"fmt"
	"net"
)

// Peer is the information for a single peer connection
type Peer struct {
	IP   net.IP
	Port uint16
}

func Unmarshal(pbs []byte) ([]Peer, error) {
	const peerSize = 6 // 4 bytes for ip, 2 for port

	// Double check that math is good
	if len(pbs)%peerSize != 0 {
		return nil, fmt.Errorf("received malphormed peers")
	}

	peerCount := len(pbs) / peerSize
	peers := make([]Peer, peerCount)
	for i := 0; i < peerCount; i++ {
		offset := i * peerSize
		peers[i].IP = net.IP(pbs[offset : offset+4])
		peers[i].Port = binary.BigEndian.Uint16(pbs[offset+4 : offset+6])
	}
	return peers, nil
}

func (p Peer) String() string {
	return net.JoinHostPort(p.IP.String(), fmt.Sprintf("%v", p.Port))
}
