package transmission

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
)

// Defines the messaging format for peer to peer communication
type MessageCode int8

const (
	MessagePing MessageCode = iota - 2
	MessagePong
	MessageRequestMetadata
	MessageMetadata
	MessageListenerSenderHandshake
	MessageListenerAcknowledgement
	MessageRequestPiece
	MessagePiece
	MessagePieceAcknowledgement
)

type PieceBlock struct {
	Index         int32
	Offset        int32
	NumTransfered int32
	Buf           []byte
}

type Message struct {
	ID MessageCode

	//Contains a sequence of bytes in this format <length><data>. Length is a type of uint32.
	//It is important to decode the payload in order lest you get bad data.
	//All variable length type except for ints have a prefix
	Payload []byte
}

// Serializes message into <size><id><payload>.
// <size> is the size of id + payload
func (m *Message) Serialize() []byte {
	length := len(m.Payload) + 1

	bytSlice := make([]byte, length+4)

	//Add size to return slice
	binary.BigEndian.PutUint32(bytSlice[0:4], uint32(length))

	//Add message Id
	bytSlice[4] = byte(m.ID)

	copy(bytSlice[5:], m.Payload)

	return bytSlice
}

func DeserializeMessage(message []byte) (*Message, error) {
	buf := bytes.NewReader(message)

	//Fetch Size
	size := make([]byte, 4)
	_, err := io.ReadFull(buf, size)
	if err != nil && err != io.EOF {
		return nil, err
	}

	msgLength := int32(binary.BigEndian.Uint32(size))

	payload := make([]byte, msgLength)
	_, err = io.ReadFull(buf, payload)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &Message{
		ID:      MessageCode(payload[0]),
		Payload: payload[1:],
	}, nil
}

func DeserializeMessageFromReader(buf io.Reader) (*Message, error) {

	//Fetch Size
	size := make([]byte, 4)
	_, err := io.ReadFull(buf, size)
	if err != nil {
		return nil, err
	}

	msgLength := int32(binary.BigEndian.Uint32(size))

	//Keep alive message
	if msgLength == 0 {
		return nil, nil
	}

	payload := make([]byte, msgLength)
	_, err = io.ReadFull(buf, payload)
	if err != nil {
		return nil, err
	}

	return &Message{
		ID:      MessageCode(payload[0]),
		Payload: payload[1:],
	}, nil
}

// Marshall Metadata into message format
func MarshallMetadata(file *Metadata) (*Message, error) {
	message := Message{ID: MessageMetadata}

	var buf bytes.Buffer

	err := writeString(&buf, file.Name)
	if err != nil {
		return nil, err
	}

	err = writeString(&buf, file.Type)
	if err != nil {
		return nil, err
	}

	err = binary.Write(&buf, binary.BigEndian, uint32(len(file.Checksum)))
	if err != nil {
		return nil, err
	}

	err = binary.Write(&buf, binary.BigEndian, file.Checksum)
	if err != nil {
		return nil, err
	}

	err = binary.Write(&buf, binary.BigEndian, uint32(file.PieceLength))
	if err != nil {
		return nil, err
	}

	err = binary.Write(&buf, binary.BigEndian, uint32(len(file.Pieces)))
	if err != nil {
		return nil, err
	}

	for i := range len(file.Pieces) {
		err = binary.Write(&buf, binary.BigEndian, file.Pieces[i])
		if err != nil {
			return nil, err
		}
	}

	err = binary.Write(&buf, binary.BigEndian, uint64(file.FileLength))
	if err != nil {
		return nil, err
	}

	message.Payload = buf.Bytes()

	return &message, nil
}

func UnmarshallMetadata(message *Message) (*Metadata, error) {
	buf := bytes.NewReader(message.Payload)
	var length uint32

	//Extract name
	err := binary.Read(buf, binary.BigEndian, &length)
	if err != nil {
		return nil, err
	}

	name := make([]byte, length)
	if _, err := buf.Read(name); err != nil {
		return nil, err
	}

	//Extract Type
	err = binary.Read(buf, binary.BigEndian, &length)
	if err != nil {
		return nil, err
	}

	mimetype := make([]byte, length)
	if _, err := buf.Read(mimetype); err != nil {
		return nil, err
	}

	//Extract checksum
	err = binary.Read(buf, binary.BigEndian, &length)
	if err != nil {
		return nil, err
	}

	checksum := make([]byte, length)
	if _, err := buf.Read(checksum); err != nil {
		return nil, err
	}

	var checksumArr [20]byte
	copy(checksumArr[:], checksum)

	//Extract Piecelength
	var pieceLength int32
	err = binary.Read(buf, binary.BigEndian, &pieceLength)
	if err != nil {
		return nil, err
	}

	//Extract pieces
	err = binary.Read(buf, binary.BigEndian, &length)
	if err != nil {
		return nil, err
	}

	pieces := make([][20]byte, length)
	for i := range length {
		var piece [20]byte

		if _, err := buf.Read(piece[:]); err != nil {
			return nil, err
		}

		copy(pieces[i][:], piece[:])
	}

	//Extract Piecelength
	var fileLength int64
	err = binary.Read(buf, binary.BigEndian, &fileLength)
	if err != nil {
		return nil, err
	}

	return &Metadata{
		Name:        string(name),
		Type:        string(mimetype),
		Checksum:    checksumArr,
		PieceLength: pieceLength,
		Pieces:      pieces,
		FileLength:  fileLength,
	}, nil
}

func MarshallPiece(file *os.File, index int) (*Message, error) {
	//Create a buf
	buf := make([]byte, PIECELENGTH)

	offset := index * PIECELENGTH

	n, err := file.ReadAt(buf, int64(offset))
	if err != nil && err != io.EOF {
		return nil, err
	}

	message := Message{ID: MessagePiece}

	//<index><offset><transfered data length><data>
	var payload bytes.Buffer
	err = binary.Write(&payload, binary.BigEndian, uint32(index))
	if err != nil {
		return nil, err
	}

	err = binary.Write(&payload, binary.BigEndian, uint32(offset))
	if err != nil {
		return nil, err
	}

	err = binary.Write(&payload, binary.BigEndian, uint32(n))
	if err != nil {
		return nil, err
	}

	message.Payload = append(payload.Bytes(), buf[:n]...)
	return &message, nil
}

func UnmarshallPiece(message *Message) (*PieceBlock, error) {
	msg := bytes.NewReader(message.Payload)
	var piece PieceBlock
	err := binary.Read(msg, binary.BigEndian, &piece.Index)
	if err != nil {
		return nil, err
	}

	err = binary.Read(msg, binary.BigEndian, &piece.Offset)
	if err != nil {
		return nil, err
	}

	err = binary.Read(msg, binary.BigEndian, &piece.NumTransfered)
	if err != nil {
		return nil, err
	}

	piece.Buf = message.Payload[12:]

	return &piece, nil
}

func RequestPiece(index int) []byte {
	msg := Message{ID: MessageRequestPiece}

	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))

	msg.Payload = payload

	return msg.Serialize()
}

func writeString(buf *bytes.Buffer, s string) error {
	b := []byte(s)

	err := binary.Write(buf, binary.BigEndian, uint32(len(b)))
	if err != nil {
		return err
	}
	_, err = buf.Write(b)
	if err != nil {
		return err
	}

	return nil
}
