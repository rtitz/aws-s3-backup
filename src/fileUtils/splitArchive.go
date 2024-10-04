package fileUtils

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/rtitz/aws-s3-backup/variables"
)

func SplitArchive(archiveFile string, SplitArchiveEachXMegaBytes int64) ([]string, error) {

	var listOfParts []string
	var bytes []byte
	var splitSize int64 = 1024 * 1024 * SplitArchiveEachXMegaBytes
	var byteCounter int64 = 0
	var partIndex int64 = 0

	f, err := os.Open(archiveFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	fileInfo, errFi := f.Stat()
	if errFi != nil {
		return listOfParts, errFi
	}

	// return without splitting if file is smaller than SplitArchiveEachXMegaBytes
	if fileInfo.Size() <= splitSize {
		listOfParts = append(listOfParts, archiveFile)
		return listOfParts, nil
	}

	br := bufio.NewReader(f)

	// infinite loop
	for {

		byteCounter++
		b, err := br.ReadByte()

		if err != nil && !errors.Is(err, io.EOF) {
			fmt.Println(err)
			break
		}

		if errors.Is(err, io.EOF) { // END OF FILE
			if bytes != nil {
				partIndex++
				fileName, err := writePartOfFile(partIndex, bytes, archiveFile)
				listOfParts = append(listOfParts, fileName)
				if err != nil {
					return listOfParts, err
				}
			}
			break
		}

		// process the one byte b
		bytes = append(bytes, b)

		if byteCounter == splitSize {
			partIndex++
			fileName, err := writePartOfFile(partIndex, bytes, archiveFile)
			listOfParts = append(listOfParts, fileName)
			bytes = nil

			if err != nil {
				return listOfParts, err
			}

			/*// READ REST OF FILE AND REWRITE IT TO SAVE SPACE DURING SPLITTING
			// Create output file
			fout, err := os.Create(archiveFile)
			if err != nil {
				log.Fatalln("Error writing archive:", err)
			}
			defer fout.Close()

			// Offset is the number of bytes you want to exclude
			_, err = f.Seek(byteCounter, io.SeekStart)
			if err != nil {
				panic(err)
			}

			n, err := io.Copy(fout, f)
			fmt.Printf("Copied %d bytes, err: %v\n", n, err)
			//os.Rename(archiveFile+"-TMP", archiveFile)
			// END OF: READ REST OF FILE AND REWRITE IT TO SAVE SPACE DURING SPLITTING*/

			byteCounter = 0
		}

		if err != nil {
			// ERROR
			fmt.Println(err)
			break
		}
	}

	return listOfParts, nil
}

func writePartOfFile(partIndex int64, bytes []byte, archiveFile string) (string, error) {
	partIndexString := fmt.Sprintf("%05d", partIndex)
	fileName := archiveFile + "-part" + partIndexString
	//fmt.Printf("\nCONTENT: %s\n%v\n", fileName, bytes)
	log.Printf("Creating part %d of %s as %s ...", partIndex, archiveFile, fileName)

	// Create output file
	out, err := os.Create(fileName)
	if err != nil {
		log.Fatalln("Error writing archive:", err)
	}
	defer out.Close()

	n, err := out.Write(bytes)
	_ = n
	if err != nil {
		panic(err)
	}
	//log.Printf("wrote %d bytes to %s\n", n, fileName)
	return fileName, nil
}

func CreateTxtHowToCombineSplittedArchive(archive string, listOfParts []string) (string, error) {
	var parts string
	var path string
	for i, part := range listOfParts {
		path = filepath.Clean(filepath.Dir(part))
		part = filepath.Base(part)
		if i == 0 { // First iteration in this loop; do not add a space in the beginning
			parts = parts + part
		} else {
			parts = parts + " " + part
		}
	}

	if !strings.HasSuffix(archive, "."+variables.ArchiveExtension) {
		archive = archive + "." + variables.ArchiveExtension
	}

	// Content is a cat command that makes cat on all files and redirects the output into a single new file
	contentOfHowToFile := fmt.Sprintf("cat %s > %s && rm -f %s %s\n", parts, archive, parts, archive+variables.HowToBuildFileSuffix)

	// Create HowToFile file
	howToFile := path + "/" + archive + variables.HowToBuildFileSuffix
	out, err := os.Create(howToFile)
	if err != nil {
		log.Fatalln("Error writing how-to-file:", err)
	}
	defer out.Close()
	w := bufio.NewWriter(out)
	w.WriteString(contentOfHowToFile)
	w.Flush()

	return howToFile, nil
}
