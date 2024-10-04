package fileUtils

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/rtitz/aws-s3-backup/cryptUtils"
	"github.com/rtitz/aws-s3-backup/variables"
)

// Decrypt downloaded files
func DecryptFiles(numberOfParts int) error {
	var encryptionSecret string = ""
	fmt.Println("\nRestored files contains encrypted files!")
	fmt.Println("Enter encryption secret (Cancel with CTRL + C):")
	fmt.Scanln(&encryptionSecret)
	fmt.Println()

	for i, part := range variables.FilesNeedingDecryption { // Iterate through the list of files to be uploaded
		partNumber := i + 1
		if partNumber > numberOfParts { // This is the HowToFile, since it is not counted as one of the archive parts
			log.Printf("Decrypting (HowToFile) ...")
		} else if numberOfParts > 1 { // Splitted archive file being uploaded
			log.Printf("Decrypting (%d/%d) ...", partNumber, numberOfParts)
		} else { // Only one (unsplitted) archive file being uploaded
			log.Printf("Decrypting...")
		}
		_, errEnc := CryptFile(false, part, "default", encryptionSecret) // True Encrypt ; False Decrypt
		if errEnc != nil {
			return errors.New("encryption failed")
		}
		//os.Remove(part) // Remove the encrypted file after decryption; Do not remove to prevent re-downloading
	}
	return nil
}

func CryptFile(encrypt bool, inputFile string, encryptionMethod string, encryptionSecret string) (string, error) {

	// TODO: Upload decryption binaries in an archive

	encryptionMethod = "default"
	_ = encryptionMethod
	encryptionSecretByte := []byte(encryptionSecret)
	var outputFile string
	//var splitEncryptionEachXMegaBytes int64 = 256 // Split each X MegaBytes encryption process. Files smaller than this are handled in one piece instead of chunks

	if encrypt {
		outputFile = inputFile + "." + variables.EncryptionExtension
	} else {
		outputFile = strings.TrimSuffix(inputFile, "."+variables.EncryptionExtension)
		outputFilePath := filepath.Dir(outputFile) + "/decrypted"
		os.MkdirAll(outputFilePath, os.ModePerm)
		fileName := filepath.Base(outputFile)
		outputFile = outputFilePath + "/" + fileName
	}

	// open input file
	fi, err := os.Open(inputFile)
	if err != nil {
		panic(err)
	}
	defer fi.Close()

	// get the size of the file
	stat, err := fi.Stat()
	if stat.Size() == 0 {
		return "", errors.New("file is empty")
	}

	// open output file
	out, err := os.Create(outputFile)
	if err != nil {
		panic(err)
	}
	defer out.Close()

	// Read entire file into memory
	var bytes []byte
	data, err := os.ReadFile(inputFile)
	if encrypt {
		bytes, _ = cryptUtils.Aes256Encrypt(encryptionSecretByte, data)
	} else {
		bytes, _ = cryptUtils.Aes256Decrypt(encryptionSecretByte, data)
		if len(bytes) == 0 {
			fmt.Println("Decrypt bytes:", len(bytes))
			os.Remove(outputFile)
			return "[ NONE ]", errors.New("wrong secret!")
		}
	}
	_, errWrite := out.Write(bytes)
	if errWrite != nil {
		panic(errWrite)
	}

	/*
		// Read file in chunks
		var bytes []byte
		var splitSize int64 = 1024 * 1024 * 256
		buf := make([]byte, splitSize)
		for {
			n, err := fi.Read(buf)
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Println(err)
				continue
			}
			if n > 0 {
				//fmt.Println(string(buf[:n]))
				if encrypt {
					bytes, _ = cryptUtils.Aes256Encrypt(encryptionSecretByte, buf[:n])
				} else {
					bytes, _ = cryptUtils.Aes256Decrypt(encryptionSecretByte, buf[:n])
					if len(bytes) == 0 {
						fmt.Println("Decrypt bytes:", len(bytes))
						os.Remove(outputFile)
						return "[ NONE ]", errors.New("wrong secret!")
					}
				}
				_, err := out.Write(bytes)
				if err != nil {
					panic(err)
				}
				bytes = nil

				if err != nil {
					return "", err
				}
			}
		}
	*/

	/*
		// Read file in chunks
		var bytes []byte
		var splitSize int64 = 1024 * 1024 * splitEncryptionEachXMegaBytes
		var byteCounter int64 = 0
		br := bufio.NewReader(fi)
		// infinite loop
		for {
			byteCounter++
			b, err := br.ReadByte()

			if err != nil && !errors.Is(err, io.EOF) {
				log.Fatal(err)
				break
			}

			if errors.Is(err, io.EOF) { // END OF FILE
				if bytes != nil {
					if encrypt {
						bytes, _ = cryptUtils.Aes256Encrypt(encryptionSecretByte, bytes)
					} else {
						bytes, _ = cryptUtils.Aes256Decrypt(encryptionSecretByte, bytes)
						if len(bytes) == 0 {
							fmt.Println("Decrypt bytes:", len(bytes))
							os.Remove(outputFile)
							return "[ NONE ]", errors.New("wrong secret!")
						}
					}
					_, err := out.Write(bytes)
					if err != nil {
						panic(err)
					}
				}
				break
			}

			// process the one byte b
			bytes = append(bytes, b)

			if byteCounter == splitSize {
				if encrypt {
					bytes, _ = cryptUtils.Aes256Encrypt(encryptionSecretByte, bytes)
				} else {
					bytes, _ = cryptUtils.Aes256Decrypt(encryptionSecretByte, bytes)
					if len(bytes) == 0 {
						fmt.Println("Decrypt bytes:", len(bytes))
						os.Remove(outputFile)
						return "[ NONE ]", errors.New("wrong secret!")
					}
				}
				_, err := out.Write(bytes)
				if err != nil {
					panic(err)
				}
				bytes = nil

				if err != nil {
					return "", err
				}

				byteCounter = 0
			}

			if err != nil {
				// ERROR
				log.Fatal(err)
				break
			}
		}
	*/

	return outputFile, nil
}
