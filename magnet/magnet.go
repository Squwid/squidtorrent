package magnet

import (
	"github.com/Squwid/squidtorrent/peers"
)

type Magnet struct {
	InfoHash [20]byte
	Name     string
	Trackers [][]string
	Peers    []peers.Peer
}

// New parses a magnet url and returns a magnet object
// func New(s string) (*Magnet, error) {
// 	uri, err := url.Parse(s)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if uri.Scheme != "magnet" {
// 		return nil, fmt.Errorf("expected scheme 'magnet' but got %v", uri.Scheme)
// 	}
// }
