package torrentfile

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/zeebo/bencode"
)

var (
	errInvalidPieceData = errors.New("invalid piece data")
	errZeroPieceLength  = errors.New("torrent has zero piece length")
	errZeroPieces       = errors.New("torrent has zero pieces")
	errPieceLength      = errors.New("piece length must be multiple of 16K")
)

const Port uint16 = 6881

// TorrentFile contains all information that a torrent needs to be downloaded
type TorrentFile struct {
	Info         TorrentInfo
	AnnounceList [][]string
	URLList      []string
}

// TorrentInfo contains info about the torrent file
type TorrentInfo struct {
	Name      string
	InfoHash  [20]byte
	Length    int64
	NumPieces uint32
	Private   bool
	Files     []File
	Info      BencodeInfo
}

type BencodeInfo struct {
	PieceLength uint32             `bencode:"piece length"`
	Pieces      []byte             `bencode:"pieces"`
	Name        string             `bencode:"name"`
	Private     bencode.RawMessage `bencode:"private"`
	Length      int64              `bencode:"length"` // Single File Mode
	Files       []file             `bencode:"files"`  // Multiple File mode
}

// File represents a file inside of a torrent
type File struct {
	Length int64
	Path   string
}

type file struct {
	Length int64    `bencode:"length"`
	Path   []string `bencode:"path"`
}

func (tf *TorrentFile) DownloadToFile(path string) error {
	// outFile, err := os.Create(path)
	// if err != nil {
	// 	return err
	// }
	// defer outFile.Close()

	// var peerID [20]byte
	// if _, err := rand.Read(peerID[:]); err != nil {
	// 	return err
	// } // TODO: Change peer id to a static one

	// peers, err := tf.requestPeers(peerID, Port)
	// if err != nil {
	// 	return err
	// }

	// torrent := p2p.Torrent{
	// 	Peers:    peers,
	// 	PeerID:   peerID,
	// 	InfoHash: tf.InfoHash,
	// 	// PieceHashes: tf.PieceHashes,
	// 	// PieceLength: tf.PieceLength,
	// 	// Length:      tf.Length,
	// 	Name: tf.Name,
	// }

	// buf, err := torrent.Download()
	// if err != nil {
	// 	return err
	// }

	// if _, err := outFile.Write(buf); err != nil {
	// 	return err
	// }
	return nil
}

// Open parses a torrent file
func Open(path string) (*TorrentFile, error) {
	var tf TorrentFile

	// Open torrent file path
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var bcode struct {
		Info         bencode.RawMessage `bencode:"info"`
		Announce     bencode.RawMessage `bencode:"announce"`
		AnnounceList bencode.RawMessage `bencode:"announce-list"`
		URLList      bencode.RawMessage `bencode:"url-list"`
	}
	if err := bencode.NewDecoder(file).Decode(&bcode); err != nil {
		return nil, err
	}
	if len(bcode.Info) == 0 {
		return nil, fmt.Errorf("expected info in torrent file but there was none")
	}

	// Info part of the encoded torrent is BencodeInfo, make a torrent object from this
	var bci BencodeInfo
	if err := bencode.DecodeBytes(bcode.Info, &bci); err != nil {
		return nil, err
	}

	ti, err := bci.toTorrent()
	if err != nil {
		return nil, err
	}

	tf.Info = *ti

	// Decide between announce list or announce url
	if len(bcode.AnnounceList) > 0 {
		var al [][]string
		if err := bencode.DecodeBytes(bcode.AnnounceList, &al); err == nil {
			for _, tier := range al {
				var ti []string
				for _, t := range tier {
					if isTrackerSupported(t) {
						ti = append(ti, t)
					}
				}
				if len(ti) > 0 {
					tf.AnnounceList = append(tf.AnnounceList, ti)
				}
			}
		}
	} else {
		var s string
		if err := bencode.DecodeBytes(bcode.Announce, &s); err == nil && isTrackerSupported(s) {
			tf.AnnounceList = append(tf.AnnounceList, []string{s})
		}
	}

	// TODO: Stuff with seeding at some point

	return &tf, nil
}

func (bci BencodeInfo) Bytes() ([]byte, error) {
	return bencode.EncodeBytes(bci)
}

func (bci BencodeInfo) hash() ([20]byte, error) {
	bs, err := bci.Bytes()
	if err != nil {
		return [20]byte{}, err
	}

	return sha1.Sum(bs), nil
}

func (ti TorrentInfo) PieceHash(index uint32) []byte {
	return ti.Info.PieceHash(index)
}

func (bci BencodeInfo) PieceHash(index uint32) []byte {
	begin := index * sha1.Size
	end := begin + sha1.Size
	return bci.Pieces[begin:end]
}

func (bci BencodeInfo) toTorrent() (*TorrentInfo, error) {
	if bci.PieceLength == 0 {
		return nil, errZeroPieceLength
	}
	if len(bci.Pieces)%sha1.Size != 0 {
		return nil, errInvalidPieceData
	}
	numPieces := len(bci.Pieces) / sha1.Size
	if numPieces == 0 {
		return nil, errZeroPieces
	}

	// No .. allowed in file names
	for _, file := range bci.Files {
		for _, path := range file.Path {
			if strings.TrimSpace(path) == ".." {
				return nil, fmt.Errorf("invalid file name %v", filepath.Join(file.Path...))
			}
		}
	}

	ti := TorrentInfo{
		NumPieces: uint32(numPieces),
		Name:      bci.Name,
		Info:      bci,
	}

	isMultiFile := len(bci.Files) > 0
	if isMultiFile {
		for _, file := range bci.Files {
			ti.Length += file.Length
		}
	} else {
		ti.Length = bci.Length
	}

	sumPiecesLength := int64(bci.PieceLength) * int64(ti.NumPieces)

	// Check that total length is not greater than piece length, since the last piece can be shorter than the rest
	if dif := sumPiecesLength - ti.Length; dif >= int64(ti.Info.PieceLength) || dif < 0 {
		return nil, errInvalidPieceData
	}

	// Hash file info
	hash, err := bci.hash()
	if err != nil {
		return nil, err
	}
	ti.InfoHash = hash

	// If name is blank, create one
	if ti.Name == "" {
		ti.Name = hex.EncodeToString(ti.InfoHash[:])
	}

	if isMultiFile {
		ti.Files = make([]File, len(bci.Files))

		for i, f := range bci.Files {
			parts := []string{clean(ti.Name)}
			for _, p := range f.Path {
				parts = append(parts, clean(p))
			}
			ti.Files[i] = File{
				Path:   filepath.Join(parts...),
				Length: f.Length,
			}
		}
	} else {
		ti.Files = []File{
			{
				Path:   clean(ti.Name),
				Length: ti.Length,
			},
		}
	}

	return &ti, nil
}

func private(b []byte) bool {
	if len(b) == 0 {
		return false
	}

	var i int64
	if err := bencode.DecodeBytes(b, &i); err != nil {
		return i != 0
	}

	var s string
	if err := bencode.DecodeBytes(b, &s); err != nil {
		return true
	}
	return !(s == "" || s == "0")
}

func isTrackerSupported(s string) bool {
	// TODO: add udp when udp is done
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") // || strings.HasPrefix(s, "udp://")
}

func clean(s string, max ...int) string {
	// Trim file name to corrent length while keeping the extension
	trim := func(s string, max int) string {
		if len(s) <= max {
			return s
		}

		ext := path.Ext(s)
		// I hope this is never the case
		if len(ext) > max {
			return s[:max]
		}

		return s[:max-len(ext)] + ext
	}

	replaceSep := func(s string) string {
		return strings.Map(func(r rune) rune {
			if r == '/' {
				return '_'
			}
			return r
		}, s)
	}

	// Default clean to 255
	var maxLength = 255
	if len(max) > 0 {
		maxLength = max[0]
	}
	s = strings.ToValidUTF8(s, string(unicode.ReplacementChar))
	s = trim(s, maxLength)
	s = strings.ToValidUTF8(s, "")

	return replaceSep(s)
}
