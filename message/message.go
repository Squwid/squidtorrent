package message

import (
	"encoding/binary"
	"fmt"
	"io"
)

// A message contains a 4 byte length, 1 byte id, and an optional payload

// The messageID describes what message that is being received
type messageID uint8

const (
	// MsgChoke chokes the receiver
	MsgChoke messageID = iota

	// MsgUnchoke unchokes the receiver
	MsgUnchoke

	// MsgInterested expresses interest in receiving data
	MsgInterested

	// MsgNotInterested expresses disinterest in receiving data
	MsgNotInterested

	// MsgHave alerts the receiver that the sender has downloaded a piece
	MsgHave

	// MsgBitfield encodes which pieces that the sender has downloaded
	MsgBitfield

	// MsgRequest requests a block of data from the receiver
	MsgRequest

	// MsgPiece delivers a block of data to fulfill a request
	MsgPiece

	// MsgCancel cancels a request
	MsgCancel
)

// Message stores the ID and payload of a message
type Message struct {
	ID      messageID
	Payload []byte
}

// FormatRequest creates a request message
func FormatRequest(index, offset, length int) *Message {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(offset))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))
	return &Message{ID: MsgRequest, Payload: payload}
}

// FormatHave creates a have message
func FormatHave(index int) *Message {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, uint32(index))
	return &Message{ID: MsgHave, Payload: payload}
}

// Serializes a message to a byte slice
// <length prefix><message ID><payload>
// Interprets `nil` as a keep-alive message
func (m *Message) Serialize() []byte {
	if m == nil {
		return make([]byte, 4)
	}
	length := uint32(len(m.Payload) + 1)         // +1 for id
	buf := make([]byte, 4+length)                // buf length is 4 (for length) + 1 (id) + ? payload
	binary.BigEndian.PutUint32(buf[0:4], length) // Put length in first 4 bytes
	buf[4] = byte(m.ID)                          // ID of message
	copy(buf[5:], m.Payload)                     // Copy in payload
	return buf
}

func (m Message) ParsePiece(index int, buf []byte) (int, error) {
	if m.ID != MsgPiece {
		return 0, fmt.Errorf("expected MsgPiece (%v), but got %v", MsgPiece, m.ID)
	}
	// Length must be 8 for index and offset
	if len(m.Payload) < 8 {
		return 0, fmt.Errorf("payload too short")
	}
	parsedIndex := int(binary.BigEndian.Uint32(m.Payload[0:4]))
	if parsedIndex != index {
		return 0, fmt.Errorf("expected index %v and got index %v", index, parsedIndex)
	}

	offset := int(binary.BigEndian.Uint32(m.Payload[4:8]))
	if offset >= len(buf) {
		return 0, fmt.Errorf("offset greater than length of payload")
	}

	data := m.Payload[8:]
	if offset+len(data) > len(buf) {
		return 0, fmt.Errorf("trying to read outside of buffer offset: %v, data: %v, buffer: %v", offset, len(data), len(buf))
	}
	copy(buf[offset:], data)
	return len(data), nil
}

func (m Message) ParseHave() (int, error) {
	if m.ID != MsgHave {
		return 0, fmt.Errorf("expected MsgHave (%v), but got %v", MsgHave, m.ID)
	}
	if len(m.Payload) != 4 {
		return 0, fmt.Errorf("expected payload length of 4 got %v", len(m.Payload))
	}
	return int(binary.BigEndian.Uint32(m.Payload)), nil
}

// Read parses a message, Returns nil on keep-alive messages
func Read(r io.Reader) (*Message, error) {
	lengthBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, lengthBuf); err != nil {
		return nil, err
	}

	// If length is 0, its a keep alive request
	lengthPayload := binary.BigEndian.Uint32(lengthBuf)
	if lengthPayload == 0 {
		return nil, nil
	}

	// Parse out 1 byte message id
	msgIDBuf := make([]byte, 1)
	if _, err := io.ReadFull(r, msgIDBuf); err != nil {
		return nil, err
	}

	// Parse payload
	msgBuf := make([]byte, lengthPayload-1)
	if _, err := io.ReadFull(r, msgBuf); err != nil {
		return nil, err
	}

	return &Message{
		ID:      messageID(msgIDBuf[0]),
		Payload: msgBuf[:],
	}, nil

}
