package transmission

// import (
// 	"bytes"
// 	"encoding/binary"
// 	"testing"
// )

// var formatPayload = []byte{0x0, 0x0, 0x0, 0xb, 0x65, 0x78, 0x61,
// 	0x6d, 0x70, 0x6c, 0x65, 0x2e, 0x74, 0x78, 0x74, 0x0, 0x0, 0x0, 0x4,
// 	0x74, 0x65, 0x78, 0x74, 0x0, 0x0, 0x0, 0x14, 0x12, 0x34, 0x56, 0x0, 0x0, 0x0, 0x0,
// 	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0,
// 	0x0, 0x0, 0x1, 0x1f, 0x8a, 0xc1, 0xf, 0x23, 0xc5, 0xb5, 0xbc, 0x11, 0x67, 0xbd, 0xa8, 0x4b, 0x83,
// 	0x3e, 0x5c, 0x5, 0x7a, 0x77, 0xd2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4, 0x0}

// func TestFormatMetadata(t *testing.T) {
// 	file := Metadata{
// 		Name:        "example.txt",
// 		Type:        "text",
// 		Checksum:    [20]byte{0x12, 0x34, 0x56},
// 		PieceLength: 262144,
// 		Pieces: [][20]byte{{0x1f, 0x8a, 0xc1, 0xf, 0x23,
// 			0xc5, 0xb5, 0xbc, 0x11, 0x67, 0xbd, 0xa8, 0x4b,
// 			0x83, 0x3e, 0x5c, 0x5, 0x7a, 0x77, 0xd2}},
// 		FileLength: 1024,
// 	}

// 	message, err := MarshallMetadata(&file)
// 	if err != nil {
// 		t.Fatalf("an error as occured while formatting file %v\n", err)
// 	}

// 	if !bytes.Equal(message.Payload, formatPayload) {
// 		t.Fatalf("invalid message payload")
// 	}

// }

// func TestFormatMetadataSerialization(t *testing.T) {
// 	file := Metadata{
// 		Name:        "example.txt",
// 		Type:        "text",
// 		Checksum:    [20]byte{0x12, 0x34, 0x56},
// 		PieceLength: 262144,
// 		Pieces: [][20]byte{{0x1f, 0x8a, 0xc1, 0xf, 0x23,
// 			0xc5, 0xb5, 0xbc, 0x11, 0x67, 0xbd, 0xa8, 0x4b,
// 			0x83, 0x3e, 0x5c, 0x5, 0x7a, 0x77, 0xd2}},
// 		FileLength: 1024,
// 	}

// 	message, err := MarshallMetadata(&file)
// 	if err != nil {
// 		t.Fatalf("an error as occured while formatting file %v\n", err)
// 	}

// 	if !bytes.Equal(message.Payload, formatPayload) {
// 		t.Fatalf("invalid message payload")
// 	}

// 	serial := message.Serialize()

// 	var size uint32
// 	if err := binary.Read(bytes.NewReader(serial), binary.BigEndian, &size); err != nil {
// 		t.Fatalf("an error as occured while formatting file %v\n", err)
// 	}

// 	expectedSize := len(formatPayload) + 1
// 	if size != uint32(expectedSize) {
// 		t.Fatalf("invalid message size")
// 	}
// }

// func TestFormatMetadataDeSerialization(t *testing.T) {
// 	file := Metadata{
// 		Name:        "example.txt",
// 		Type:        "text",
// 		Checksum:    [20]byte{0x12, 0x34, 0x56},
// 		PieceLength: 262144,
// 		Pieces: [][20]byte{{0x1f, 0x8a, 0xc1, 0xf, 0x23,
// 			0xc5, 0xb5, 0xbc, 0x11, 0x67, 0xbd, 0xa8, 0x4b,
// 			0x83, 0x3e, 0x5c, 0x5, 0x7a, 0x77, 0xd2}},
// 		FileLength: 1024,
// 	}

// 	message, err := MarshallMetadata(&file)
// 	if err != nil {
// 		t.Fatalf("an error as occured while formatting file %v\n", err)
// 	}

// 	if !bytes.Equal(message.Payload, formatPayload) {
// 		t.Fatalf("invalid message payload")
// 	}

// 	serial := message.Serialize()

// 	var size uint32
// 	if err := binary.Read(bytes.NewReader(serial), binary.BigEndian, &size); err != nil {
// 		t.Fatalf("an error as occured while formatting file %v\n", err)
// 	}

// 	expectedSize := len(formatPayload) + 1
// 	if size != uint32(expectedSize) {
// 		t.Fatalf("invalid message size")
// 	}

// 	msg, err := DeserializeMessage(serial)
// 	if err != nil {
// 		t.Fatalf("an error as occured while deserialing the message %v\n", err)
// 	}

// 	if msg.ID != message.ID {
// 		t.Fatalf("invalid message id")
// 	}
// }

// func TestFormatMetadataParsing(t *testing.T) {
// 	file := Metadata{
// 		Name:        "example.txt",
// 		Type:        "text",
// 		Checksum:    [20]byte{0x12, 0x34, 0x56},
// 		PieceLength: 262144,
// 		Pieces: [][20]byte{{0x1f, 0x8a, 0xc1, 0xf, 0x23,
// 			0xc5, 0xb5, 0xbc, 0x11, 0x67, 0xbd, 0xa8, 0x4b,
// 			0x83, 0x3e, 0x5c, 0x5, 0x7a, 0x77, 0xd2}},
// 		FileLength: 1024,
// 	}

// 	message, err := MarshallMetadata(&file)
// 	if err != nil {
// 		t.Fatalf("an error as occured while formatting file %v\n", err)
// 	}

// 	if !bytes.Equal(message.Payload, formatPayload) {
// 		t.Fatalf("invalid message payload")
// 	}

// 	serial := message.Serialize()

// 	var size uint32
// 	if err := binary.Read(bytes.NewReader(serial), binary.BigEndian, &size); err != nil {
// 		t.Fatalf("an error as occured while formatting file %v\n", err)
// 	}

// 	expectedSize := len(formatPayload) + 1
// 	if size != uint32(expectedSize) {
// 		t.Fatalf("invalid message size")
// 	}

// 	msg, err := DeserializeMessage(serial)
// 	if err != nil {
// 		t.Fatalf("an error as occured while deserialing the message %v\n", err)
// 	}

// 	if msg.ID != message.ID {
// 		t.Fatalf("invalid message id")
// 	}

// 	newFile, err := UnmarshallMetadata(message)
// 	if err != nil {
// 		t.Fatalf("an error as occured while parsing the message %v\n", err)
// 	}

// 	if file.Name != newFile.Name {
// 		t.Fatalf("invalid file name got %s wanted %s", newFile.Name, file.Name)
// 	}

// 	if file.Type != newFile.Type {
// 		t.Fatalf("invalid file type got %s wanted %s", newFile.Type, file.Type)
// 	}

// 	if file.Checksum != newFile.Checksum {
// 		t.Fatalf("invalid file checksum got %s wanted %s", newFile.Checksum, file.Checksum)
// 	}

// 	if file.PieceLength != newFile.PieceLength {
// 		t.Fatalf("invalid file piece length got %d wanted %d", newFile.PieceLength, file.PieceLength)
// 	}

// 	if len(file.Pieces) != len(newFile.Pieces) {
// 		t.Fatalf("pieces mismatch got(%d) wanted (%d)", len(file.Pieces), len(newFile.Pieces))
// 	}

// 	for i := range newFile.Pieces {
// 		if !bytes.Equal(newFile.Pieces[i][:], file.Pieces[i][:]) {
// 			t.Fatalf("invalid piece")
// 		}
// 	}

// 	if file.FileLength != newFile.FileLength {
// 		t.Fatalf("invalid file length got %d wanted %d", newFile.FileLength, file.FileLength)
// 	}
// }

// func TestMarshallPiece(t *testing.T) {
// 	path := "./testdata/TCP-IP.pdf"
// 	file, err := os.Open(path)
// 	if err != nil {
// 		t.Fatalf("an error has occured: %v\n", err)
// 	}

// 	defer file.Close()

// 	msg, err := MarshallPiece(file, 2)
// 	if err != nil {
// 		t.Fatalf("an error has occured: %v\n", err)
// 	}

// 	if len(msg.Payload) != PIECELENGTH+12 {
// 		t.Fatalf("invalid payload size wanted %d got %d", PIECELENGTH+12, len(msg.Payload))
// 	}
// }

// func TestMarshallAltered(t *testing.T) {
// 	path := "./testdata/TCP-IP.pdf"
// 	file, err := os.Open(path)
// 	if err != nil {
// 		t.Fatalf("an error has occured: %v\n", err)
// 	}

// 	defer file.Close()

// 	msg, err := MarshallPiece(file, 2)
// 	if err != nil {
// 		t.Fatalf("an error has occured: %v\n", err)
// 	}

// 	if len(msg.Payload) != PIECELENGTH+12 {
// 		t.Fatalf("invalid payload size wanted %d got %d", PIECELENGTH+12, len(msg.Payload))
// 	}

// 	msgSame, err := MarshallPiece(file, 2)
// 	if err != nil {
// 		t.Fatalf("an error has occured: %v\n", err)
// 	}

// 	if !bytes.Equal(msg.Payload, msgSame.Payload) {
// 		t.Fatalf("error seperate payloads %d %d", len(msg.Payload), len(msgSame.Payload))
// 	}
// }

// func TestMarshallCorrect(t *testing.T) {
// 	path := "./testdata/TCP-IP.pdf"
// 	file, err := os.Open(path)
// 	if err != nil {
// 		t.Fatalf("an error has occured: %v\n", err)
// 	}

// 	defer file.Close()

// 	msg, err := MarshallPiece(file, 2)
// 	if err != nil {
// 		t.Fatalf("an error has occured: %v\n", err)
// 	}

// 	if len(msg.Payload) != PIECELENGTH+12 {
// 		t.Fatalf("invalid payload size wanted %d got %d", len(msg.Payload), PIECELENGTH+12)
// 	}

// 	piece, err := UnmarshallPiece(msg)
// 	if err != nil {
// 		t.Fatalf("an error has occured: %v\n", err)
// 	}

// 	metadata, _, err := GenerateMetadata(path)
// 	if err != nil {
// 		t.Fatalf("an error has occured: %v\n", err)
// 	}

// 	reqPieceHash := metadata.Pieces[2]

// 	pieceBufHash := sha1.Sum(piece.Buf)

// 	if !bytes.Equal(reqPieceHash[:], pieceBufHash[:]) {
// 		t.Fatalf("error piece is not valid")
// 	}
// }
