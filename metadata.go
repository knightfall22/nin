package transmission

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/gabriel-vasile/mimetype"
)

const PIECELENGTH = 512 * 1024 // 512 KB

type FileInfo struct {
	Path              string
	Size              int64
	CummulativeOffset int64
	AbsolutePath      string
}

type Metadata struct {
	Name        string
	Type        string
	Checksum    [20]byte
	PieceLength int32
	Pieces      [][20]byte
	FileLength  int64
	Single      bool
	Folders     []FileInfo
}

// Generate metadata from file
func GenerateMetadata(path string) (*Metadata, *VirtualFile, error) {
	vf := VirtualFile{
		rootPath: path,
	}

	if err := vf.Build(); err != nil {
		return nil, nil, err
	}

	metadata := vf.ToMetadata()
	metadata.FileLength = vf.totalSize
	metadata.Single = vf.single

	return metadata, &vf, nil

}

func generateFileMetadata(path string) (*Metadata, error) {
	var Metadata Metadata

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	Metadata.Name = filepath.Base(f.Name())

	//Retrieve file length
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	Metadata.FileLength = stat.Size()

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

type VirtualFile struct {
	rootPath  string
	files     []FileInfo
	handles   []*os.File
	pieces    [][20]byte
	totalSize int64
	single    bool

	//used for building path to write to
	downloadPath string
}

func (vf *VirtualFile) findFileAndOffset(globalOffset int64) (fileIndex int, localOffset int64) {
	if globalOffset < 0 {
		return 0, 0
	}

	if globalOffset >= vf.totalSize {
		if len(vf.files) > 0 {
			lastFile := len(vf.files) - 1
			return lastFile, vf.files[lastFile].Size
		}

		return 0, 0
	}

	fileIndex = sort.Search(len(vf.files), func(i int) bool {
		return vf.files[i].CummulativeOffset+vf.files[i].Size > globalOffset
	})

	if fileIndex < len(vf.files) {
		localOffset = globalOffset - vf.files[fileIndex].CummulativeOffset
	}

	return fileIndex, localOffset
}

func (vf *VirtualFile) ReadAt(p []byte, offset int64) (int, error) {
	// Find starting file
	fileIndex, localOffset := vf.findFileAndOffset(offset)

	bytesRead := 0

	for len(p) > 0 && fileIndex < len(vf.files) {
		n, err := vf.handles[fileIndex].ReadAt(p, localOffset)

		bytesRead += n
		p = p[n:]

		//handle transition to next file
		if err == io.EOF && fileIndex < len(vf.files)-1 {
			fileIndex++
			localOffset = 0
			continue
		}

		if err != nil && err != io.EOF {
			return bytesRead, err
		}
		break
	}

	return bytesRead, nil

}

func (vf *VirtualFile) WriteAt(offset int64, p []byte) (int, error) {
	// Find starting file
	fileIndex, localOffset := vf.findFileAndOffset(offset)

	bytesWritten := 0

	for len(p) > 0 && fileIndex < len(vf.files) {
		if vf.handles[fileIndex] == nil {
			var path string
			fileBase := filepath.Base(vf.rootPath)

			if vf.single {
				path = filepath.Join(vf.downloadPath, fileBase)
			} else {
				path = filepath.Join(vf.downloadPath, fileBase, vf.files[fileIndex].Path)
			}

			fmt.Println("path", vf.downloadPath)
			fmt.Println("dir", filepath.Dir(path))

			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				panic(err)
			}

			file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
			if err != nil {
				return bytesWritten, err
			}

			vf.handles[fileIndex] = file
		}

		file := vf.handles[fileIndex]

		maxWriteSize := vf.files[fileIndex].Size - localOffset
		if maxWriteSize <= 0 {
			fileIndex++
			localOffset = 0
			continue
		}

		writeSize := int64(len(p))
		writeSize = min(writeSize, maxWriteSize)

		n, err := file.WriteAt(p[:writeSize], localOffset)
		if err != nil {
			return bytesWritten, err
		}

		p = p[n:]

		bytesWritten += n
		fileIndex++
		localOffset = 0
	}

	return bytesWritten, nil

}

func (vf *VirtualFile) Build() error {
	info, err := os.Stat(vf.rootPath)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		vf.single = true
	}

	err = filepath.Walk(vf.rootPath, vf.walkFunc)
	if err != nil {
		return err
	}

	sort.Slice(vf.files, func(i, j int) bool {
		return vf.files[i].Path < vf.files[j].Path
	})

	vf.calculateCummulativeOffsets()

	if err := vf.buildFileHandles(); err != nil {
		return err
	}

	pieces, err := vf.generatePieces()
	if err != nil {
		return err
	}

	vf.pieces = pieces

	return nil
}

func (vf *VirtualFile) ToMetadata() *Metadata {
	var metadata Metadata

	metadata.Name = vf.rootPath
	metadata.PieceLength = PIECELENGTH
	metadata.Folders = vf.files
	metadata.Pieces = vf.pieces

	return &metadata
}

func (vf *VirtualFile) Close() error {
	for _, file := range vf.handles {
		if err := file.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (vf *VirtualFile) buildFileHandles() error {
	for _, f := range vf.files {
		open, err := os.Open(f.AbsolutePath)
		if err != nil {
			return err
		}

		vf.handles = append(vf.handles, open)
	}

	return nil
}

func (vf *VirtualFile) generatePieces() ([][20]byte, error) {
	//Hash 512kb blocks of file
	numPieces := (vf.totalSize + PIECELENGTH - 1) / PIECELENGTH

	pieces := make([][20]byte, numPieces)

	buf := make([]byte, PIECELENGTH)

	for i := range numPieces {
		offset := i * PIECELENGTH
		n, err := vf.ReadAt(buf, int64(offset))
		if err != nil && err != io.EOF {
			return nil, err
		}

		hash := sha1.Sum(buf[:n])
		pieces[i] = hash
	}

	return pieces, nil
}

func (vf *VirtualFile) calculateCummulativeOffsets() {
	var offset int64

	for i := range vf.files {
		vf.files[i].CummulativeOffset = offset
		offset += vf.files[i].Size
	}
}

func (vf *VirtualFile) walkFunc(path string, info fs.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if info.IsDir() {
		if path == vf.rootPath {
			return nil
		}
	}

	relative, err := filepath.Rel(vf.rootPath, path)
	if err != nil {
		return err
	}

	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	if info.Size() == 0 {
		return nil
	}

	if !info.IsDir() {
		fileInfo := FileInfo{
			Path:         relative,
			AbsolutePath: absolute,
			Size:         info.Size(),
		}
		vf.files = append(vf.files, fileInfo)
		vf.totalSize += fileInfo.Size
	}

	return nil
}
