package torrentfile

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"math/rand"
	"os"

	"github.com/Squwid/squidtorrent/p2p"
	"github.com/jackpal/bencode-go"
)

const Port uint16 = 6881

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

func (tf *TorrentFile) DownloadToFile(path string) error {
	var peerID [20]byte
	if _, err := rand.Read(peerID[:]); err != nil {
		return err
	} // TODO: Change peer id to a static one

	peers, err := tf.requestPeers(peerID, Port)
	if err != nil {
		return err
	}

	torrent := p2p.Torrent{
		Peers:       peers,
		PeerID:      peerID,
		InfoHash:    tf.InfoHash,
		PieceHashes: tf.PieceHashes,
		PieceLength: tf.PieceLength,
		Length:      tf.Length,
		Name:        tf.Name,
	}
	buf, err := torrent.Download()
	if err != nil {
		return err
	}

	outFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer outFile.Close()

	if _, err := outFile.Write(buf); err != nil {
		return err
	}
	return nil
}

// Open parses a torrent file
func Open(path string) (TorrentFile, error) {
	var t = TorrentFile{}

	file, err := os.Open(path)
	if err != nil {
		return t, err
	}
	defer file.Close()

	var bto bencodeTorrent
	if err := bencode.Unmarshal(file, &bto); err != nil {
		return t, err
	}
	return bto.toTorrent()
}

func (bto bencodeTorrent) toTorrent() (TorrentFile, error) {
	var t = TorrentFile{}
	infoHash, err := bto.Info.hash()
	if err != nil {
		return t, err
	}
	pieceHashes, err := bto.Info.splitPieceHashes()
	if err != nil {
		return t, err
	}

	return TorrentFile{
		Announce:    bto.Announce,
		InfoHash:    infoHash,
		PieceHashes: pieceHashes,
		PieceLength: bto.Info.PieceLength,
		Length:      bto.Info.Length,
		Name:        bto.Info.Name,
	}, nil
}

func (bi bencodeInfo) hash() ([20]byte, error) {
	var buf bytes.Buffer
	if err := bencode.Marshal(&buf, bi); err != nil {
		return [20]byte{}, err
	}
	return sha1.Sum(buf.Bytes()), nil
}

func (bi bencodeInfo) splitPieceHashes() ([][20]byte, error) {
	buf := []byte(bi.Pieces)
	if len(buf)%20 != 0 {
		return nil, fmt.Errorf("got bad bencode pieces length of %v", len(buf))
	}

	hashCount := len(buf) / 20
	hashes := make([][20]byte, hashCount)

	for i := 0; i < hashCount; i++ {
		copy(hashes[i][:], buf[i*20:(i+1)*20])
	}
	return hashes, nil
}
