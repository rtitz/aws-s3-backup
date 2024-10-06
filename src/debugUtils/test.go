package debugUtils

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/rtitz/aws-s3-backup/cryptUtils"
	"github.com/rtitz/aws-s3-backup/variables"
)

// Playground function for tests
func Test() {
	var memStatus runtime.MemStats
	start_time := time.Now()

	//Piece of code
	testFile := "/Users/rene/tmp/enc-test/hangouts-takeout-20240703T062159Z-001.tar"
	//testFile = "/Volumes/ramdisk_1g/tmp/go.mod"
	//testFile = "/Volumes/ramdisk_1g/tmp/2024-03-12-raspios-bookworm-arm64-lite.img.xz"

	//fmt.Println("Encrypt: ", testFile)
	//outputFileEnc, errEnc := fileUtils.CryptFile(true, testFile, variables.EncryptionAlgorithm, "test123") // True Encrypt ; False Decrypt
	//fmt.Println("Encryption:", outputFileEnc, errEnc)
	//fmt.Println()
	fmt.Println("Decrypt: ", testFile+".enc")
	outputFileDec, errDec := cryptUtils.CryptFile(false, testFile+".enc", variables.EncryptionAlgorithm, "test123") // True Encrypt ; False Decrypt
	fmt.Println("Decryption:", outputFileDec, errDec)

	//fileUtils.CryptFile(true, "/Volumes/ramdisk_1g/tmp/pico2.tar.gz", variables.EncryptionAlgorithm, "test123")
	//fileUtils.CryptFile(true, "/Volumes/ramdisk_1g/tmp/2024-03-12-raspios-bookworm-arm64-lite.img.xz", variables.EncryptionAlgorithm, "test123") // True Encrypt ; False Decrypt
	//fileUtils.CryptFile(false, "/Volumes/ramdisk_1g/tmp/2024-03-12-raspios-bookworm-arm64-lite.img.xz", variables.EncryptionAlgorithm, "test123") // True Encrypt ; False Decrypt

	runtime.ReadMemStats(&memStatus)
	duration := time.Since(start_time)

	info := fmt.Sprintf("Elapsed time = %s. Total memory(MB) consumed = %v", duration, memStatus.Sys/1024/1024)
	fmt.Println("\n" + info)
	os.Exit(0)
}
