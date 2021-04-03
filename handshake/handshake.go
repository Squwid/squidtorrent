package handshake

import (
	"fmt"
	"io"
)

// A Handshake is a special message that a peer uses to identify itself
type Handshake struct {
	Pstr     string
	InfoHash [20]byte
	PeerID   [20]byte
}

// New creates a new handshake with the standard pstr
func New(infoHash, peerID [20]byte) *Handshake {
	return &Handshake{
		Pstr:     "BitTorrent protocol",
		InfoHash: infoHash,
		PeerID:   peerID,
	}
}

func (hs Handshake) Serialize() []byte {
	buf := make([]byte, len(hs.Pstr)+49) // 1 + 19 + 8 + 20 + 20
	buf[0] = byte(len(hs.Pstr))          // Length of 'BitTorrent protocol' which is 19 so 0x13

	// i is current index
	i := 1
	i += copy(buf[i:], []byte(hs.Pstr))
	i += copy(buf[i:], make([]byte, 8)) // 8 empty bytes, until extensions!
	i += copy(buf[i:], hs.InfoHash[:])  // Requested file hash
	i += copy(buf[i:], hs.PeerID[:])    // squidtorrent's peer id

	return buf
}

// Read parses an incoming handshake, rather than serializing one
func Read(r io.Reader) (*Handshake, error) {
	protocolLength := make([]byte, 1) // First byte is length of protocol
	if _, err := io.ReadFull(r, protocolLength); err != nil {
		return nil, err
	}

	// Check if more than 0 (really needs to be 19)
	pstrLen := int(protocolLength[0])
	if pstrLen == 0 {
		return nil, fmt.Errorf("ptrlen cannot be 0")
	}

	// Parse rest of handshake excluding first byte
	handshakeBuf := make([]byte, 48+pstrLen) // Length of protocol + empty 8 bytes (8) + Infohash(20) + PeerID(20)
	if _, err := io.ReadFull(r, handshakeBuf); err != nil {
		return nil, err
	}

	var infoHash, peerID [20]byte

	copy(infoHash[:], handshakeBuf[pstrLen+8:pstrLen+8+20])
	copy(peerID[:], handshakeBuf[pstrLen+8+20:])

	return &Handshake{
		Pstr:     string(handshakeBuf[0:pstrLen]),
		InfoHash: infoHash,
		PeerID:   peerID,
	}, nil
}
