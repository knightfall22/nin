package transmission

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gabriel-vasile/mimetype"
)

const PIECELENGTH = 512 * 1024 // 512 KB

//Todo: might use this later
// type FileType int8

// const (
// 	DIR FileType = iota
// 	FILE
// )

type Metadata struct {
	Name        string
	Type        string
	Checksum    [20]byte
	PieceLength int32
	Pieces      [][20]byte
	FileLength  int64
}

// Generate metadata from file
func GenerateMetadata(path string) (*Metadata, error) {
	var Metadata Metadata
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	Metadata.Name = filepath.Base(f.Name())

	fmt.Println(Metadata.Name)

	//Retrieve file length
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	Metadata.FileLength = info.Size()

	//Set piece length(512Kib)
	Metadata.PieceLength = PIECELENGTH

	Metadata.Pieces, err = pieceFile(f, Metadata.FileLength)
	if err != nil {
		return nil, err
	}

	//Fetch mimetype
	mime, file, err := recycleReader(f)
	if err != nil {
		return nil, err
	}

	Metadata.Type = mime

	//Get checksum
	hasher := sha1.New()

	_, err = io.Copy(hasher, file)
	if err != nil {
		return nil, err
	}

	fullhash := hasher.Sum(nil)
	copy(Metadata.Checksum[:], fullhash)

	return &Metadata, nil
}

// Read file mimetype and return read bytes
func recycleReader(input io.Reader) (mimeType string, recycled io.Reader, err error) {
	header := bytes.NewBuffer(nil)

	mtype, err := mimetype.DetectReader(io.TeeReader(input, header))
	if err != nil {
		return
	}

	recycled = io.MultiReader(header, input)

	return mtype.String(), recycled, nil
}

func pieceFile(input *os.File, size int64) ([][20]byte, error) {
	//Hash 512kb blocks of file
	numPieces := (size + PIECELENGTH - 1) / PIECELENGTH

	pieces := make([][20]byte, numPieces)

	buf := make([]byte, PIECELENGTH)

	for i := range numPieces {
		offset := i * PIECELENGTH
		n, err := input.ReadAt(buf, int64(offset))
		if err != nil && err != io.EOF {
			return nil, err
		}

		hash := sha1.Sum(buf[:n])
		copy(pieces[i][:], hash[:])
	}

	return pieces, nil
}

//[178 0 77 65 39 107 104 32 11 176 13 70 69 130 237 92 112 242 119 79]
