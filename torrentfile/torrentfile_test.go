package torrentfile

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

// var update = flag.Bool("update", false, "update .golden.json files")

func TestTorrent(t *testing.T) {
	tor, err := Open("data_test/ubuntu-14.04.1-server-amd64.iso.torrent")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "ubuntu-14.04.1-server-amd64.iso", tor.Info.Name)
	assert.Equal(t, int64(599785472), tor.Info.Length)
	assert.Equal(t, "2d066c94480adcf52bfd1185a75eb4ddc1777673", hex.EncodeToString(tor.Info.InfoHash[:]))
	assert.Equal(t, [][]string{
		{"http://torrent.ubuntu.com:6969/announce"},
		{"http://ipv6.torrent.ubuntu.com:6969/announce"},
	}, tor.AnnounceList)
}

// func TestToTorrent(t *testing.T) {
// 	tests := map[string]struct {
// 		input  BencodeInfo
// 		output *TorrentInfo
// 		fails  bool
// 	}{}
// }

// func TestToTorrentFile(t *testing.T) {
// 	tests := map[string]struct {
// 		input  *bencodeTorrent
// 		output TorrentFile
// 		fails  bool
// 	}{
// 		"correct conversion": {
// 			input: &bencodeTorrent{
// 				Announce: "http://bttracker.debian.org:6969/announce",
// 				Info: bencodeInfo{
// 					Pieces:      "1234567890abcdefghijabcdefghij1234567890",
// 					PieceLength: 262144,
// 					Length:      351272960,
// 					Name:        "debian-10.2.0-amd64-netinst.iso",
// 				},
// 			},
// 			output: TorrentFile{
// 				Announce: "http://bttracker.debian.org:6969/announce",
// 				InfoHash: [20]byte{216, 247, 57, 206, 195, 40, 149, 108, 204, 91, 191, 31, 134, 217, 253, 207, 219, 168, 206, 182},
// 				PieceHashes: [][20]byte{
// 					{49, 50, 51, 52, 53, 54, 55, 56, 57, 48, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106},
// 					{97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 49, 50, 51, 52, 53, 54, 55, 56, 57, 48},
// 				},
// 				PieceLength: 262144,
// 				Length:      351272960,
// 				Name:        "debian-10.2.0-amd64-netinst.iso",
// 			},
// 			fails: false,
// 		},
// 		"not enough bytes in pieces": {
// 			input: &bencodeTorrent{
// 				Announce: "http://bttracker.debian.org:6969/announce",
// 				Info: bencodeInfo{
// 					Pieces:      "1234567890abcdefghijabcdef", // Only 26 bytes
// 					PieceLength: 262144,
// 					Length:      351272960,
// 					Name:        "debian-10.2.0-amd64-netinst.iso",
// 				},
// 			},
// 			output: TorrentFile{},
// 			fails:  true,
// 		},
// 	}

// 	for _, test := range tests {
// 		to, err := test.input.toTorrent()
// 		if test.fails {
// 			assert.NotNil(t, err)
// 		} else {
// 			assert.Nil(t, err)
// 		}
// 		assert.Equal(t, test.output, to)
// 	}
// }
