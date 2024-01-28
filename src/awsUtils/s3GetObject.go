package awsUtils

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func GetObject(ctx context.Context, cfg aws.Config, bucket, object, downloadLocation string) (bool, error) {

	clientS3 := s3.NewFromConfig(cfg)
	if !strings.HasSuffix(downloadLocation, "/") {
		downloadLocation = downloadLocation + "/"
	}
	fileName := downloadLocation + object
	dirOfFile := filepath.Dir(fileName)

	if err := os.MkdirAll(dirOfFile, os.ModePerm); err != nil {
		log.Fatalln("Error creating destination directorty:", err)
	}

	if _, err := os.Stat(fileName); err == nil {
		return false, nil
	}

	result, err := clientS3.GetObject(ctx, &s3.GetObjectInput{Bucket: &bucket, Key: &object})
	if err != nil {
		log.Printf("Couldn't get object! %v\n", err)
		return false, err
	}
	defer result.Body.Close()
	file, err := os.Create(fileName)
	if err != nil {
		log.Printf("Couldn't create file %v / %v\n", fileName, err)
		return false, err
	}
	defer file.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		log.Printf("Couldn't read object body from %v. / %v\n", object, err)
	}
	_, err = file.Write(body)
	return true, err
}
