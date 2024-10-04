package fileUtils

import (
	"fmt"
	"strconv"
)

// Calculates the file size units of a float64 number
func FileSizeUnitCalculation(sizeInByte float64) (float64, float64, string) {
	//sizeRaw := float64(stat.Size())
	sizeRaw := sizeInByte
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
	}

	return sizeRaw, size, unit
}
