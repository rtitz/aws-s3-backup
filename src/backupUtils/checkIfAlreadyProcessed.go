package backupUtils

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

/*
checkIfPathAlreadyProcessed checks if the path is already processed by checking the processedTrackingFile.
If write is true, it creates a new entry in the processedTrackingFile with the current timestamp and the number of parts.
If write is false, it checks if the path is already present in the processedTrackingFile.
*/
func checkIfPathAlreadyProcessed(processedTrackingFile, path string, listOfParts []string, write bool) (bool, error) {
	if write { // Create processed file, if not existing and add a new path to file
		out, err := os.OpenFile(processedTrackingFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalln("Error writing 'processed' file:", err)
		}
		defer out.Close()
		w := bufio.NewWriter(out)
		dt := time.Now()
		timestampStr := dt.Format(time.RFC1123)
		timestampUnixStr := strconv.Itoa(int(dt.Unix()))

		if len(listOfParts) > 1 { // Splitted means HowTo exists as additional element in the listOfParts
			listOfParts = listOfParts[:len(listOfParts)-1]
		}
		numberOfParts := len(listOfParts)

		stringToWrite := path + "\n * Timestamp of upload: " + timestampUnixStr + " (" + timestampStr + ")\n * Number of file parts: " + strconv.Itoa(numberOfParts)
		w.WriteString(stringToWrite + "\n\n")
		w.Flush()
		out.Close()
		return true, nil
	} else { // Check if file already processed
		processed := false
		if _, err := os.Stat(processedTrackingFile); errors.Is(err, os.ErrNotExist) {
			//fmt.Println("file not exist")
			return processed, nil
		}
		readFile, err := os.Open(processedTrackingFile)
		if err != nil {
			fmt.Println(err)
		}
		defer readFile.Close()
		fileScanner := bufio.NewScanner(readFile)
		fileScanner.Split(bufio.ScanLines)
		for fileScanner.Scan() {
			//fmt.Println(fileScanner.Text())
			if fileScanner.Text() == path {
				processed = true
				break
			}
		}
		readFile.Close()
		return processed, nil
	}
}
