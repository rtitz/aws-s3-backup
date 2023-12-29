package awsUtils

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/rtitz/aws-s3-backup/variables"
)

func PutObject(ctx context.Context, cfg aws.Config, file, bucket, object string, storageClass types.StorageClass) error {

	// https://aws.github.io/aws-sdk-go-v2/docs/sdk-utilities/s3/

	f, errF := os.Open(file)
	if errF != nil {
		log.Println("Failed opening file", file, errF)
	}
	defer f.Close()

	/*
		_, sizeRaw, size, unit, sha256checksum, err := fileUtils.GetFileInfo(file)
		_ = sizeRaw
		_ = sha256checksum

		if err != nil {
			log.Fatalf("ERROR: %v\n", err)
		}

		//sha256checksumInBase64 := base64.StdEncoding.EncodeToString([]byte(sha256checksum))
		log.Printf("Upload %s (%.2f %s) to %s key: %s ... \n", file, size, unit, bucket, object)
		//log.Println("Pre-calculated checksum: ", sha256checksum, sha256checksumInBase64)
	*/

	clientS3 := s3.NewFromConfig(cfg)
	//output, err := clientS3.PutObject(ctx, &s3.PutObjectInput{Bucket: &bucket, Key: &object, StorageClass: types.StorageClassStandard, Body: f})

	if variables.UploadMethod == "TransferManager" {
		//uploader := manager.NewUploader(clientS3)
		uploader := manager.NewUploader(clientS3, func(u *manager.Uploader) {
			u.PartSize = variables.SplitUploadsEachXMegaBytes * 1024 * 1024 // 64MB per part
		})
		output, err := uploader.Upload(context.TODO(), &s3.PutObjectInput{
			Bucket:       aws.String(bucket),
			Key:          aws.String(object),
			Body:         f,
			StorageClass: storageClass,
			//ChecksumSHA256:    aws.String(sha256checksumInBase64),
			ChecksumAlgorithm: types.ChecksumAlgorithmSha256,
		})
		if err != nil {
			log.Fatalf("FAILED S3TransferManager upload %v\n", err)
		}
		_ = output
		//log.Println(*output.ChecksumSHA256)
	} else if variables.UploadMethod == "PutObject" {
		output, err := clientS3.PutObject(ctx, &s3.PutObjectInput{
			Bucket:            &bucket,
			Key:               &object,
			Body:              f,
			StorageClass:      storageClass,
			ChecksumAlgorithm: types.ChecksumAlgorithmSha256,
		})
		if err != nil {
			log.Fatalf("FAILED PutObject %v\n", err)
		}
		_ = output
	} else if variables.UploadMethod == "Disabled" {
		time.Sleep(time.Millisecond * 3000)
		log.Println("Upload Disabled!")
		return nil
	}

	//log.Printf("UPLOAD DONE!\n")
	return nil
}
