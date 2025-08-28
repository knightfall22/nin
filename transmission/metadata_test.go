package transmission

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
)

func TestGenerateMetadata(t *testing.T) {
	res, _, err := GenerateMetadata("./testdata/TCP-IP.pdf")
	if err != nil {
		t.Fatalf("an error as occured while generating metadata %v\n", err)
	}

	message, err := MarshallMetadata(res)
	if err != nil {
		t.Fatalf("an error as occured while generating metadata %v\n", err)
	}

	fi, err := UnmarshallMetadata(message)
	if err != nil {
		t.Fatalf("an error as occured while generating metadata %v\n", err)
	}

	if fi.Name != res.Name ||
		fi.Type != res.Type ||
		fi.Checksum != res.Checksum || fi.PieceLength != res.PieceLength ||
		fi.FileLength != res.FileLength {
		t.Fatalf("invalid metadata")
	}

	if len(fi.Pieces) != len(res.Pieces) {
		t.Fatalf("pieces mismatch got(%d) wanted (%d)", len(fi.Pieces), len(res.Pieces))
	}

	for i := range res.Pieces {
		if !bytes.Equal(res.Pieces[i][:], fi.Pieces[i][:]) {
			t.Fatalf("invalid piece")
		}
	}
}

func TestReadingVirtualFile(t *testing.T) {
	_, vf, err := GenerateMetadata("./testdata/small")
	if err != nil {
		t.Fatalf("an error as occured while generating metadata %v\n", err)
	}

	fileList := []string{
		"./testdata/small/example1.pdf",
		"./testdata/small/example2.pdf",
		"./testdata/small/example3.pdf",
		"./testdata/small/example4.pdf",
		"./testdata/small/example5.pdf",
		"./testdata/small/example6.pdf",
	}

	var offset int64
	for _, v := range fileList {
		info, err := os.Stat(v)
		if err != nil {
			t.Fatal(err)
		}

		//Open file
		f, err := os.Open(v)
		if err != nil {
			t.Fatal(err)
		}
		buf1 := make([]byte, info.Size())
		if _, err := f.Read(buf1); err != nil {
			t.Fatal(err)
		}

		buf2 := make([]byte, info.Size())
		vf.ReadAt(buf2, offset)
		offset += info.Size()

		if !bytes.Equal(buf1, buf2) {
			fmt.Printf("expected data in virtual file to be the same with file")
		}

		f.Close()
	}

	vf.Close()
}

func TestWritingFromVirtualFile(t *testing.T) {
	PIECELENGTH = 150 * 1024 //150kb
	meta, vf, err := GenerateMetadata("./testdata/small/")
	if err != nil {
		t.Fatalf("an error as occured while generating metadata %v\n", err)
	}

	lf := VirtualFile{
		rootPath:     meta.Name,
		downloadPath: "./testdata/result",
		files:        meta.Folders,
		pieces:       meta.Pieces,
		totalSize:    meta.FileLength,
		handles:      make([]*os.File, len(meta.Folders)),
		single:       meta.Single,
	}

	fileList1 := []string{
		"./testdata/result/small/example1.pdf",
		"./testdata/result/small/example2.pdf",
		"./testdata/result/small/example3.pdf",
		"./testdata/result/small/example4.pdf",
		"./testdata/result/small/example5.pdf",
		"./testdata/result/small/example6.pdf",
	}

	fileList2 := []string{
		"./testdata/small/example1.pdf",
		"./testdata/small/example2.pdf",
		"./testdata/small/example3.pdf",
		"./testdata/small/example4.pdf",
		"./testdata/small/example5.pdf",
		"./testdata/small/example6.pdf",
	}

	for i := range vf.pieces {
		offset := i * PIECELENGTH

		buf := make([]byte, PIECELENGTH)

		if _, err := vf.ReadAt(buf, int64(offset)); err != nil && err != io.EOF {
			t.Fatalf("failed while reading %v\n", err)
		}

		_, err := lf.WriteAt(int64(offset), buf)
		if err != nil {
			t.Fatal(err)
		}
	}

	mb := 1024 * 1024

	for i, v := range fileList1 {
		//Open file
		f, err := os.Open(v)
		if err != nil {
			t.Fatal(err)
		}
		buf1 := make([]byte, mb)
		n, err := f.Read(buf1)
		if err != nil {
			t.Fatal(err)
		}

		buf1 = buf1[:n]

		f, err = os.Open(fileList2[i])
		if err != nil {
			t.Fatal(err)
		}

		buf2 := make([]byte, mb)
		n, err = f.Read(buf2)
		if err != nil {
			t.Fatal(err)
		}

		buf2 = buf2[:n]

		if !bytes.Equal(buf1, buf2) {
			fmt.Printf("expected data in virtual file to be the same with file")
		}

		f.Close()
	}

	vf.Close()
	lf.Close()
	os.RemoveAll("./testdata/result/small")
}
