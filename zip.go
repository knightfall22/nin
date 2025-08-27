package transmission

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

var ErrNotFolder = fmt.Errorf("cannot zip single file")

// Zip all files in provide path and return path to zip folder
func ZipFolder(destination string, source string) (string, error) {
	info, err := os.Stat(source)
	if err != nil {
		return "", err
	}

	if !info.IsDir() {
		return "", ErrNotFolder
	}

	// Ensure the zip directory exists
	if err := os.MkdirAll(destination, 0755); err != nil {
		return "", err
	}

	out := fmt.Sprintf("%s.zip", filepath.Base(source))
	destination = filepath.Join(destination, out)

	archive, err := os.OpenFile(destination, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Hello")
		return "", err
	}

	w := zip.NewWriter(archive)
	err = filepath.Walk(source, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == "." {
			return nil
		}

		if info.Mode().IsRegular() {
			f1, err := os.Open(path)
			if err != nil {
				return err
			}

			defer f1.Close()

			w1, err := w.Create(f1.Name())
			if err != nil {
				return err
			}

			if _, err := io.Copy(w1, f1); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	err = w.Close()
	return destination, err
}
