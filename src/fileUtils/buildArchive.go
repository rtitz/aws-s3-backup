package fileUtils

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/rtitz/aws-s3-backup/variables"
)

// Handles the creation of an archive file and adds the files of the given file list to that archive
func BuildArchive(files []string, archiveFile string) (string, error) {
	archiveFile = archiveFile + "." + variables.ArchiveExtension
	archiveFile = strings.ReplaceAll(archiveFile, " ", "-") // REPLACE SPACE WITH -

	log.Println("Creating archive...")
	// Create output file
	out, err := os.Create(archiveFile)
	if err != nil {
		log.Fatalln("Error writing archive:", err)
	}
	defer out.Close()

	// Create the archive and write the output to the "out" Writer
	var keepArchiveFile bool
	keepArchiveFile, err = createArchive(files, out)
	if err != nil {
		out.Close()
		os.Remove(archiveFile)
		log.Fatalln("Error creating archive:", err)
	}
	if keepArchiveFile {
		log.Println("Archive created successfully")
	} else { // Archive not created since it is already an archive
		os.Remove(archiveFile)
		archiveFile = strings.TrimSuffix(archiveFile, "."+variables.ArchiveExtension)

		file := files[0]
		source, err := os.Open(file) //open the source file
		if err != nil {
			panic(err)
		}
		defer source.Close()

		destination, err := os.Create(archiveFile) //create the destination file
		if err != nil {
			panic(err)
		}
		defer destination.Close()
		_, err = io.Copy(destination, source) //copy the contents of source to destination file
		if err != nil {
			panic(err)
		}
		log.Println("Existing archive copied successfully")
	}
	return archiveFile, nil
}

// Creates an archive file
func createArchive(files []string, buf io.Writer) (bool, error) {
	// Create new Writers for gzip and tar
	// These writers are chained. Writing to the tar writer will
	// write to the gzip writer which in turn will write to
	// the "buf" writer
	gw := gzip.NewWriter(buf)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Iterate over files and add them to the tar archive
	for _, file := range files {
		file := filepath.Clean(file)

		// Do not create archive if it is already an archive
		if strings.HasSuffix(file, "."+variables.ArchiveExtension) && len(files) == 1 {
			log.Printf("%s is already a %s archive. - Skip build archive, copy instead...\n", file, variables.ArchiveExtension)
			return false, nil
		}

		os.Chdir(filepath.Dir(file))
		//fmt.Println("PWD:", filepath.Dir(file))
		//fmt.Println("BASE:", filepath.Base(file))

		// This was needed before Go 1.22
		err := addToArchive(tw, filepath.Base(file))
		if err != nil {
			return false, err
		}

		// Since Go 1.22 the following is possible, but will not output the singe files added to archive.
		// tw.AddFS(os.DirFS(file))

	}
	return true, nil
}

/*
addToArchive will check if path to be added is a directory or a file.
A file will be added (through calling addFileToArchive)
A directory will be walked through and the files inside will be added (through calling addFileToArchive)
*/
func addToArchive(tw *tar.Writer, path string) error {

	// Open the file which will be written into the archive
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get FileInfo about our file providing file size, mode, etc.
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	if !fileInfo.IsDir() {
		addFileToArchive(tw, file, fileInfo, path)
	}

	if fileInfo.IsDir() {
		err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Println(err)
				return err
			}
			if !info.IsDir() {
				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()
				fileInfo, err := file.Stat()
				if err != nil {
					return err
				}
				addFileToArchive(tw, file, fileInfo, path)
			}
			return nil
		})
		if err != nil {
			fmt.Println(err)
		}
	}
	return nil
}

// Adds given files to the archive
func addFileToArchive(tw *tar.Writer, file *os.File, fileInfo fs.FileInfo, filename string) error {
	log.Printf("Add %s to archive...\n", filename)
	// Create a tar Header from the FileInfo data
	header, err := tar.FileInfoHeader(fileInfo, fileInfo.Name())
	if err != nil {
		return err
	}

	// Use full path as name (FileInfoHeader only takes the basename)
	// If we don't do this the directory strucuture would
	// not be preserved
	// https://golang.org/src/archive/tar/common.go?#L626
	header.Name = filename

	// Write file header to the tar archive
	err = tw.WriteHeader(header)
	if err != nil {
		return err
	}

	// Copy file content to tar archive
	_, err = io.Copy(tw, file)
	if err != nil {
		return err
	}
	return nil
}
