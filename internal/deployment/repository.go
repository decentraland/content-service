package deployment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/decentraland/content-service/internal/entities"
)

const proofPrefix = "proof/"
const parcelsPrefix = "parcels/"

type Repository interface {
	StoreMapping(id string, parcels []string) error
	StoreDeployment(d *entities.Deploy, p *entities.DeployProof) error
}

type Config struct {
	Bucket string
	ACL    string
	URL    string
}

func NewRepository(c *Config) Repository {
	return &deployRepo{
		Bucket: c.Bucket,
		ACL:    c.ACL,
		URL:    c.URL,
	}

}

type deployRepo struct {
	Bucket string
	ACL    string
	URL    string
}

func (mr *deployRepo) StoreMapping(id string, parcels []string) error {
	sess := session.Must(session.NewSession())

	uploader := &s3manager.Uploader{
		S3:                s3.New(sess, aws.NewConfig().WithEndpoint(mr.URL)),
		PartSize:          s3manager.DefaultUploadPartSize,
		Concurrency:       s3manager.DefaultUploadConcurrency,
		LeavePartsOnError: false,
		MaxUploadParts:    s3manager.MaxUploadParts,
	}

	strings.NewReader(id)

	for _, p := range parcels {
		if err := mr.storeFile(uploader, p, strings.NewReader(id), "text/plain"); err != nil {
			return err
		}
	}

	l := strings.NewReader(strings.Join(parcels, "|"))
	if err := mr.storeFile(uploader, fmt.Sprintf("%s%s", parcelsPrefix, id), l, "text/plain"); err != nil {
		return err
	}

	return nil
}

func (mr *deployRepo) StoreDeployment(d *entities.Deploy, p *entities.DeployProof) error {
	sess := session.Must(session.NewSession())

	uploader := &s3manager.Uploader{
		S3:                s3.New(sess, aws.NewConfig().WithEndpoint(mr.URL)),
		PartSize:          s3manager.DefaultUploadPartSize,
		Concurrency:       s3manager.DefaultUploadConcurrency,
		LeavePartsOnError: false,
		MaxUploadParts:    s3manager.MaxUploadParts,
	}

	deploy, err := json.Marshal(d)
	if err != nil {
		return err
	}

	if err := mr.storeFile(uploader, p.ID, bytes.NewReader(deploy), "application/json"); err != nil {
		return err
	}

	proof, err := json.Marshal(p)
	if err != nil {
		return err
	}

	pKey := fmt.Sprintf("%s%s", proofPrefix, p.ID)
	if err := mr.storeFile(uploader, pKey, bytes.NewReader(proof), "application/json"); err != nil {
		return err
	}

	return nil
}

func (mr *deployRepo) storeFile(u *s3manager.Uploader, key string, content io.Reader, cType string) error {
	_, err := u.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(mr.Bucket),
		Key:         aws.String(key),
		ACL:         aws.String(mr.ACL),
		Body:        content,
		ContentType: aws.String(cType),
	})

	if err != nil {
		return err
	}

	return nil
}
