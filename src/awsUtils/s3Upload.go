package awsUtils

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/rtitz/aws-s3-backup/fileUtils"
	"github.com/rtitz/aws-s3-backup/variables"
)

func PutObject(ctx context.Context, cfg aws.Config, file, bucket, object string, storageClass types.StorageClass) error {

	// https://aws.github.io/aws-sdk-go-v2/docs/sdk-utilities/s3/

	f, errF := os.Open(file)
	if errF != nil {
		log.Println("Failed opening file", file, errF)
	}
	defer f.Close()

	// Checksum mode
	var checksumMode string
	if variables.ChecksumMode == "sha256" {
		checksumMode = "sha256"
	} else if variables.ChecksumMode == "md5" {
		checksumMode = "md5"
	}

	// Get file info
	_, sizeRaw, size, unit, checksum, err := fileUtils.GetFileInfo(file, checksumMode)
	_ = sizeRaw
	_ = size
	_ = unit

	if err != nil {
		log.Fatalf("ERROR GETTING FILE INFO: %v\n", err)
	}
	//sha256checksumInBase64 := base64.StdEncoding.EncodeToString([]byte(sha256checksum))
	//log.Printf("Upload %s (%.2f %s) to %s key: %s ... \n", file, size, unit, bucket, object)
	//log.Println("Pre-calculated checksum: ", sha256checksum, sha256checksumInBase64)

	//localChecksumBase64 := fmt.Sprintf("%q\n", base64.StdEncoding.EncodeToString([]byte(checksum)))

	clientS3 := s3.NewFromConfig(cfg)
	//output, err := clientS3.PutObject(ctx, &s3.PutObjectInput{Bucket: &bucket, Key: &object, StorageClass: types.StorageClassStandard, Body: f})

	if variables.UploadMethod == "TransferManager" {
		//uploader := manager.NewUploader(clientS3)
		uploader := manager.NewUploader(clientS3, func(u *manager.Uploader) {
			u.PartSize = variables.SplitUploadsEachXMegaBytes * 1024 * 1024 // 64MB per part
		})

		params := s3.PutObjectInput{}
		if checksumMode == "sha256" {
			params = s3.PutObjectInput{
				Bucket:       aws.String(bucket),
				Key:          aws.String(object),
				Body:         f,
				StorageClass: storageClass,
				//ChecksumSHA256:    aws.String(sha256checksumInBase64),
				ChecksumAlgorithm: types.ChecksumAlgorithmSha256,
			}
		} else if checksumMode == "md5" {
			params = s3.PutObjectInput{
				Bucket:       aws.String(bucket),
				Key:          aws.String(object),
				Body:         f,
				StorageClass: storageClass,
			}
		}

		output, err := uploader.Upload(context.TODO(), &params)
		if err != nil {
			log.Fatalf("FAILED S3TransferManager upload %v\n", err)
		}
		_ = output

	} else if variables.UploadMethod == "PutObject" {
		params := s3.PutObjectInput{}
		if checksumMode == "sha256" {
			params = s3.PutObjectInput{
				Bucket:            &bucket,
				Key:               &object,
				Body:              f,
				StorageClass:      storageClass,
				ChecksumAlgorithm: types.ChecksumAlgorithmSha256,
				//ChecksumSHA256:    &localSha256checksumBase64,
			}
		} else if checksumMode == "md5" {
			params = s3.PutObjectInput{
				Bucket:       &bucket,
				Key:          &object,
				Body:         f,
				StorageClass: storageClass,
			}
		}

		output, err := clientS3.PutObject(ctx, &params)
		if err != nil {
			log.Fatalf("FAILED PutObject %v\n", err)
		}

		// Verify checksum
		if checksumMode == "sha256" {
			uploadedObjectChecksum := *output.ChecksumSHA256
			//uploadedObjectChecksumDecoded, _ := base64.StdEncoding.DecodeString(uploadedObjectChecksum)

			if uploadedObjectChecksum == checksum {
				log.Printf("Checksum %s : OK", uploadedObjectChecksum)
			} else {
				log.Printf("CHECKSUM FAIL! - Checksum of uploaded object: %s / Checksum of local file: %s\n", uploadedObjectChecksum, checksum)
				return errors.New("CHECKSUM FAIL")
			}
		} else if checksumMode == "md5" {
			uploadedObjectChecksum := *output.ETag
			uploadedObjectChecksum = strings.TrimLeft(uploadedObjectChecksum, "\"")
			uploadedObjectChecksum = strings.TrimRight(uploadedObjectChecksum, "\"")
			if uploadedObjectChecksum == checksum {
				log.Printf("Checksum %s : OK", uploadedObjectChecksum)
			} else {
				log.Printf("CHECKSUM FAIL! - Checksum of uploaded object: %s / Checksum of local file: %s\n", uploadedObjectChecksum, checksum)
				return errors.New("CHECKSUM FAIL")
			}
		}

	} else if variables.UploadMethod == "Disabled" {
		time.Sleep(time.Millisecond * 2000)
		log.Println("Upload Disabled!")
		return nil
	}

	return nil
}
