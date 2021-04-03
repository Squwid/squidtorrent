package torrentfile

import (
	"io"

	"github.com/jackpal/bencode-go"
)

// TorrentFile contails metadata from a .torrent file
type TorrentFile struct {
	Announce    string
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
}

type bencodeInfo struct {
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
}

type bencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     bencodeInfo `bencode:"info"`
}

// Open parses a torrent file
func Open(r io.Reader) (*bencodeTorrent, error) {
	var bto = bencodeTorrent{}
	if err := bencode.Unmarshal(r, &bto); err != nil {
		return nil, err
	}
	return &bto, nil
}

func (bto bencodeTorrent) torrentFile() (*TorrentFile, error) {
	return nil, nil
}
