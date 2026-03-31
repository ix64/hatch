package storage

import "github.com/ix64/s3-go/s3common"

type Config struct {
	Enabled              bool
	Endpoint             string
	Bucket               string
	BucketLookup         s3common.BucketLookupType
	Prefix               string
	Region               string
	AccessKey            string
	SecretKey            string
	PresignExpireSeconds int
}
