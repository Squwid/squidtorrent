package p2p

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"time"

	"github.com/Squwid/squidtorrent/client"
	"github.com/Squwid/squidtorrent/message"
	"github.com/Squwid/squidtorrent/peers"
	"github.com/sirupsen/logrus"
)

// MaxBlockSize is the largest number of bytes a request can ask for (16KiB).
// Clients are supposed to sever connections that ask for a size this big, but try to increase it anyways
const MaxBlockSize = 16384

// MaxBacklog is the number of unfulfilled requests a client can have in its pipeline
// TODO: Tweak this for better download speeds
const MaxBacklog = 5

// Torrent contains data to download a torrent from a list of peers
type Torrent struct {
	Peers       []peers.Peer
	PeerID      [20]byte   // This client identifier
	InfoHash    [20]byte   // File that we need, seeder must have entire file to download
	PieceHashes [][20]byte // Hash of each individual file piece (usually more than 16KB)
	PieceLength int
	Length      int
	Name        string
}

/*
	A block is a broken down piece, since a piece can be > 16KiB, a block is a part of a piece.
	A block is what gets requested and is MaxBlockSize
*/

// pieceWork is what gets sent to the worker to receive a part of the torrent
type pieceWork struct {
	index  int
	hash   [20]byte
	length int
}

// pieceResult is the result of a piece that is sent through the result channel from the worker
type pieceResult struct {
	index int
	buf   []byte
}

// pieceProgress tracks the progress of getting different blocks and combining them to a piece
type pieceProgress struct {
	index      int
	client     *client.Client
	buf        []byte
	downloaded int
	requested  int
	backlog    int
}

func (t *Torrent) startDownloader(peer peers.Peer, workChan chan *pieceWork, resultsChan chan *pieceResult, l *logrus.Entry) {
	if l == nil {
		l = &logrus.Entry{}
	}
	l = l.WithField("Peer", peer.IP)

	// Create peer connection
	c, err := client.New(peer, t.PeerID, t.InfoHash)
	if err != nil {
		l.WithError(err).Errorf("Could not establish connection with peer")
		return
	}
	defer c.Conn.Close()
	l.Debugf("Successfully completed handshake")

	if err := c.SendUnchoked(); err != nil {
		l.WithError(err).Errorf("Error sending unchoked to peer")
		return
	}

	if err := c.SendInterested(); err != nil {
		l.WithError(err).Errorf("Error sending interested to peer")
		return
	}

	// Read from workQueue when possible
	for pw := range workChan {
		// Peer doenst have piece, stick back on chan
		if !c.Bitfield.HasPiece(pw.index) {
			workChan <- pw
			continue
		}

		buf, err := downloadPiece(c, pw)
		if err != nil {
			l.WithError(err).Errorf("Errror downloading piece")
			workChan <- pw
			return
		}

		if err := checkIntegrity(pw, buf); err != nil {
			l.WithError(err).Errorf("Failed integrity check")
			workChan <- pw
			continue
		}

		c.SendHave(pw.index)
		resultsChan <- &pieceResult{
			index: pw.index,
			buf:   buf,
		}
	}
}

func checkIntegrity(pw *pieceWork, buf []byte) error {
	hash := sha1.Sum(buf)
	if !bytes.Equal(hash[:], pw.hash[:]) {
		return fmt.Errorf("index %v failed integrity check", pw.index)
	}
	return nil
}

func (state *pieceProgress) readMsg() error {
	// This call blocks until peer sends a message. If timeout is hit, peer is dropped and requeue'd
	msg, err := state.client.Read()
	if err != nil {
		return err
	}

	// Keep alive
	if msg == nil {
		return nil
	}

	switch msg.ID {
	case message.MsgUnchoke:
		state.client.Choked = false
	case message.MsgChoke:
		state.client.Choked = true
	case message.MsgHave:
		index, err := msg.ParseHave()
		if err != nil {
			return err
		}
		state.client.Bitfield.SetPiece(index)

	case message.MsgPiece:
		n, err := msg.ParsePiece(state.index, state.buf)
		if err != nil {
			return err
		}
		state.downloaded += n
		state.backlog--
	}
	return nil
}

// downloadPiece gets all blocks and combines them to a piece
func downloadPiece(c *client.Client, pw *pieceWork) ([]byte, error) {
	state := pieceProgress{
		index:  pw.index,
		client: c,
		buf:    make([]byte, pw.length),
	}

	// 30 second timeout, incase peer has slow as shit internet.
	// Disable the deadline after successful download
	c.Conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer c.Conn.SetDeadline(time.Time{})

	for state.downloaded < pw.length {
		if !state.client.Choked {
			for state.backlog < MaxBacklog && state.requested < pw.length {
				blockSize := MaxBlockSize

				// Handle last block
				if pw.length-state.requested < blockSize {
					blockSize = pw.length - state.requested
				}

				// Give us the data!
				if err := c.SendRequest(pw.index, state.requested, blockSize); err != nil {
					return nil, err
				}
				state.backlog++
				state.requested += blockSize
			}
		}

		if err := state.readMsg(); err != nil {
			return nil, err
		}
	}
	return state.buf, nil
}

// pieceBounds gets the piece length. all pieces will be the same except the end piece
func (t *Torrent) pieceBounds(index int) (begin int, end int) {
	begin = index * t.PieceLength
	end = begin + t.PieceLength

	if end > t.Length {
		end = t.Length
	}
	return
}

func (t *Torrent) pieceSize(index int) int {
	begin, end := t.pieceBounds(index)
	return end - begin
}

// Download downloads the torrent. This stores the entire file in memory.
func (t *Torrent) Download() ([]byte, error) {
	logger := logrus.WithField("Name", t.Name)
	logger.Infof("Starting download")

	// Initialize channels
	workChan := make(chan *pieceWork, len(t.PieceHashes))
	resultsChan := make(chan *pieceResult)
	for i, hash := range t.PieceHashes {
		length := t.pieceSize(i)
		workChan <- &pieceWork{
			index:  i,
			hash:   hash,
			length: length,
		}
	}

	// get to fucking work
	for _, peer := range t.Peers {
		go t.startDownloader(peer, workChan, resultsChan, logger)
	}

	// TODO: Change to file store instead of mem store
	buf := make([]byte, t.Length)
	donePieces := 0
	for donePieces < len(t.PieceHashes) {
		res := <-resultsChan
		begin, end := t.pieceBounds(res.index)
		copy(buf[begin:end], res.buf)
		donePieces++

		percent := float64(donePieces) / float64(len(t.PieceHashes)) * 100
		logger.WithFields(logrus.Fields{
			"Percent":      fmt.Sprintf("%0.2f%%", percent),
			"Piece":        res.index,
			"Total Pieces": len(t.PieceHashes),
		}).Infof("Downloaded piece")
	}
	close(workChan)

	return buf, nil
}
