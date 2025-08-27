package transmission

import (
	"bytes"
	"fmt"
	"testing"
)

func TestGenerateMetadata(t *testing.T) {
	res, _, err := GenerateMetadata("./testdata/example.jpg")
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

func TestMe(t *testing.T) {
	_, vf, err := GenerateMetadata("./testdata/books")
	if err != nil {
		t.Fatalf("an error as occured while generating metadata %v\n", err)
	}

	fmt.Printf("Virtual file %+v\n", vf)
}
