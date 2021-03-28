package torrentfile

import (
	"fmt"
	"net/url"
)

type bencodeTrackerResp struct {
	Interval int `bencode:"interval"` // How often to reconnect to the tracker to refresh list of peers (in seconds)

	// Peers is a blob that contains ip addresses of each peer, by groups of 6 bytes, (first 4 ip, last 2 port)
	Peers string `bencode:"peers"`
}

// Build GET request url to hit tracker to announce presense as a peer and receeive list of other peers
func (tf *TorrentFile) buildTrackerURL(peerID [20]byte, port uint16) (string, error) {
	base, err := url.Parse(tf.Announce)
	if err != nil {
		return "", err
	}

	params := url.Values{
		"info_hash":  []string{string(tf.InfoHash[:])}, // Identifies the file that is gonna get downloaded
		"peer_id":    []string{string(peerID[:])},      // Real BitTorrent clients have pre-generated ids, come up with our own
		"port":       []string{fmt.Sprintf("%v", port)},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{fmt.Sprintf("%v", tf.Length)},
	}

	// Craft up the url with the values
	base.RawQuery = params.Encode()
	return base.String(), nil
}

// func r(tf *TorrentFile) requestPeers(peerID [20]byte, port uint16)
