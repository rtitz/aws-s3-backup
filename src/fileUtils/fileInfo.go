package fileUtils

import (
	"bufio"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
)

// Get info / stats and checksums about a file
func GetFileInfo(file, checksumMode string) (*os.File, float64, float64, string, string, error) {

	f, errF := os.Open(file)
	if errF != nil {
		log.Println("Failed opening file", file, errF)
	}
	defer f.Close()

	// Get the file size
	stat, err := f.Stat()
	if err != nil {
		fmt.Println(err)
		return nil, 0, 0, "", "", err
	}

	// Read the file into a byte slice
	bsOfFile := make([]byte, stat.Size())
	_, err = bufio.NewReader(f).Read(bsOfFile)
	if err != nil && err != io.EOF {
		fmt.Println(err)
		return nil, 0, 0, "", "", err
	}

	// Checksum
	var checksum string
	if checksumMode == "sha256" {
		// SHA256 Checksum of File
		h := sha256.New()
		h.Write(bsOfFile)
		bs := h.Sum(nil)
		//sha256checksum := fmt.Sprintf("%x", bs)
		//checksum = sha256checksum
		checksum = string(base64.StdEncoding.EncodeToString(bs))

	} else if checksumMode == "md5" {
		// MD5 Checksum of File
		h := md5.New()
		h.Write(bsOfFile)
		bs := h.Sum(nil)
		//md5checksum := hex.EncodeToString(h.Sum(nil))
		md5checksum := fmt.Sprintf("%x", bs)
		checksum = md5checksum
	}

	sizeRaw, size, unit := FileSizeUnitCalculation(float64(stat.Size()))

	return f, sizeRaw, size, unit, checksum, nil
}
