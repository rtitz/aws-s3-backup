package fileUtils

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
)

func GetFileInfo(file string) (*os.File, float64, float64, string, string, error) {

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

	// SHA256 Checksum of File
	h := sha256.New()
	h.Write(bsOfFile)
	//bs := h.Sum(nil)
	sha256checksum := hex.EncodeToString(h.Sum(nil))
	//sha256checksum := fmt.Sprintf("%x\n", bs)

	sizeRaw := float64(stat.Size())
	size := sizeRaw
	unit := "B"

	kb := float64(sizeRaw) / (1 << 10) // KB
	kbR := fmt.Sprintf("%.2f", kb)
	mb := float64(sizeRaw) / (1 << 20) // MB
	mbR := fmt.Sprintf("%.2f", mb)
	gb := float64(sizeRaw) / (1 << 30) // GB
	gbR := fmt.Sprintf("%.2f", gb)
	tb := float64(sizeRaw) / (1 << 40) // TB
	tbR := fmt.Sprintf("%.2f", tb)
	pb := float64(sizeRaw) / (1 << 50) // PB
	pbR := fmt.Sprintf("%.2f", pb)

	if value, _ := strconv.ParseFloat(pbR, 64); value >= 1 {
		size = value
		unit = "PB"
	} else if value, _ := strconv.ParseFloat(tbR, 64); value >= 1 {
		size = value
		unit = "TB"
	} else if value, _ := strconv.ParseFloat(gbR, 64); value >= 1 {
		size = value
		unit = "GB"
	} else if value, _ := strconv.ParseFloat(mbR, 64); value >= 1 {
		size = value
		unit = "MB"
	} else if value, _ := strconv.ParseFloat(kbR, 64); value >= 1 {
		size = value
		unit = "KB"
	} else if value, _ := strconv.ParseFloat(kbR, 64); value < 1 {
		size = value
		unit = "B"
	}

	return f, sizeRaw, size, unit, sha256checksum, nil
}
