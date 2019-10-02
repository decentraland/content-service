package metrics

import (
	"fmt"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	log "github.com/sirupsen/logrus"
	"gopkg.in/segmentio/analytics-go.v3"
)

type ddClient interface {
	RecordBytesStored(fileSize int64)
	RecordBytesRetrieved(fileSize int64)
	RecordRetrieveTime(t time.Duration)
	RecordStorageTime(t time.Duration)
	RecordUploadReqSize(size int)
	RecordUploadRequestFiles(files int)
	RecordManifestSize(size int)
	RecordUploadProcessTime(t time.Duration)
	RecordUploadRequestParseTime(t time.Duration)
	RecordIsMemberTime(t time.Duration)
	RecordDCLResponseTime(t time.Duration)
	RecordUploadRequestValidationTime(t time.Duration)
	RecordGetParcelMetadata(t time.Duration)
	RecordGetParcelContent(t time.Duration)
	RecordStoreContent(t time.Duration)
	RecordStoreMetadata(t time.Duration)
	RecordDCLAPIError(status int)
}

type segmentClient interface {
	RecordUpload(uploadId string, uploader string, parcels []string, files map[string][]string, origin string)
}

type Agent struct {
	ddClient
	segmentClient
}

type ddClientImpl struct {
	client *statsd.Client
	tags   []string
}

func (c *ddClientImpl) gauge(metric string, value float64) {
	if err := c.client.Gauge(fmt.Sprintf(".%s", metric), value, c.tags, 1); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (c *ddClientImpl) RecordBytesStored(fileSize int64) {
	c.gauge("FileStored.bytes", float64(fileSize))
}

func (c *ddClientImpl) RecordBytesRetrieved(fileSize int64) {
	c.gauge("FileRetrieved.bytes", float64(fileSize))
}

func (c *ddClientImpl) RecordUploadRequestValidationTime(t time.Duration) {
	c.gauge("UploadValidationTime.msec.call", toMillis(t))
}

func (c *ddClientImpl) RecordRetrieveTime(t time.Duration) {
	c.gauge("StorageDownloadTime.msec.call", toMillis(t))
}

func (c *ddClientImpl) RecordStorageTime(t time.Duration) {
	c.gauge("StorageTime.msec.call", toMillis(t))
}

func (c *ddClientImpl) RecordDCLResponseTime(t time.Duration) {
	c.gauge("DCLResponseTime.msec.call", toMillis(t))
}

func (c *ddClientImpl) RecordUploadRequestFiles(files int) {
	c.gauge("UploadedRequestFiles", float64(files))
}

func (c *ddClientImpl) RecordManifestSize(size int) {
	c.gauge("ManifestRequestEntries", float64(size))
}

func (c *ddClientImpl) RecordUploadProcessTime(t time.Duration) {
	c.gauge("UploadProcessTime.msec.call", toMillis(t))
}

func (c *ddClientImpl) RecordUploadRequestParseTime(t time.Duration) {
	c.gauge("UploadParseTime.msec.call", toMillis(t))
}

func (c *ddClientImpl) RecordUploadReqSize(size int) {
	c.gauge("UploadRequestSize.bytes", float64(size))
}

func (c *ddClientImpl) RecordIsMemberTime(t time.Duration) {
	c.gauge("IsMemberDuration.msec.call", toMillis(t))
}

func (c *ddClientImpl) RecordGetParcelMetadata(t time.Duration) {
	c.gauge("GetParcelMetadata.msec.call", toMillis(t))
}

func (c *ddClientImpl) RecordGetParcelContent(t time.Duration) {
	c.gauge("GetParcelContent.msec.call", toMillis(t))
}

func (c *ddClientImpl) RecordStoreContent(t time.Duration) {
	c.gauge("StoreContent.msec.call", toMillis(t))
}

func (c *ddClientImpl) RecordStoreMetadata(t time.Duration) {
	c.gauge("StoreMetadata.msec.call", toMillis(t))
}

func (c *ddClientImpl) RecordDCLAPIError(status int) {
	c.gauge(fmt.Sprintf("DecentralandAPIError%d", status), float64(1))
}

type ddClientDummy struct{}

func (d *ddClientDummy) RecordBytesStored(fileSize int64)                  {}
func (d *ddClientDummy) RecordBytesRetrieved(fileSize int64)               {}
func (d *ddClientDummy) RecordUploadRequestValidationTime(t time.Duration) {}
func (d *ddClientDummy) RecordRetrieveTime(t time.Duration)                {}
func (d *ddClientDummy) RecordUploadReqSize(size int)                      {}
func (d *ddClientDummy) RecordDCLResponseTime(t time.Duration)             {}
func (d *ddClientDummy) RecordUploadRequestFiles(files int)                {}
func (d *ddClientDummy) RecordManifestSize(size int)                       {}
func (d *ddClientDummy) RecordUploadProcessTime(t time.Duration)           {}
func (d *ddClientDummy) RecordUploadRequestParseTime(t time.Duration)      {}
func (d *ddClientDummy) RecordIsMemberTime(t time.Duration)                {}
func (d *ddClientDummy) RecordStorageTime(t time.Duration)                 {}
func (d *ddClientDummy) RecordGetParcelMetadata(t time.Duration)           {}
func (d *ddClientDummy) RecordGetParcelContent(t time.Duration)            {}
func (d *ddClientDummy) RecordStoreContent(t time.Duration)                {}
func (d *ddClientDummy) RecordStoreMetadata(t time.Duration)               {}
func (d *ddClientDummy) RecordDCLAPIError(status int)                      {}

type segmentClientImpl struct {
	client analytics.Client
}

func (sa *segmentClientImpl) RecordUpload(uploadId string, uploader string, parcels []string, files map[string][]string, origin string) {

	filesData := []contentOcurrence{}
	for hash, paths := range files {
		filesData = append(filesData, contentOcurrence{Hash: hash, Files: paths})
	}

	err := sa.client.Enqueue(analytics.Track{
		UserId: uploader,
		Event:  "Content Upload",
		Properties: analytics.NewProperties().
			Set("files", filesData).
			Set("parcels", parcels).
			Set("cid", uploadId).
			Set("origin", origin),
	})
	if err != nil {
		log.Errorf("[SEGMENT] Failed to queue event : %s", err.Error())
	}
}

type segmentDummy struct {
	client analytics.Client
}

func (sa *segmentDummy) RecordUpload(uploadId string, uploader string, parcels []string, files map[string][]string, origin string) {
}

type Config struct {
	Enabled      bool
	AppName      string
	AnalyticsKey string
}

func Make(metrics Config) (*Agent, error) {

	var ddClient ddClient
	if metrics.Enabled {
		c, err := buildDDClient(metrics.AppName)
		if err != nil {
			return nil, err
		}
		ddClient = c
	} else {
		ddClient = &ddClientDummy{}
	}

	segAgent, err := buildSegmentAgent(metrics.AnalyticsKey)
	if err != nil {
		return nil, err
	}

	return &Agent{ddClient, segAgent}, nil
}

func buildDDClient(appName string) (ddClient, error) {

	log.Debug("Initializing DD Agent")
	client, err := statsd.New("", statsd.WithNamespace(appName))
	if err != nil {
		return nil, err
	}

	log.Infof("Newrelic agent loaded for: %s", appName)

	return &ddClientImpl{client: client, tags: []string{}}, nil
}

func buildSegmentAgent(writeKey string) (segmentClient, error) {
	log.Debug("Initializing Segment Agent")
	if len(writeKey) == 0 {
		log.Debug("No Segment configuration found")
		return &segmentDummy{}, nil
	}

	c := analytics.New(writeKey)

	log.Info("Segment client initialized")
	return &segmentClientImpl{client: c}, nil
}

func toMillis(d time.Duration) float64 {
	return float64(d.Nanoseconds() / int64(time.Millisecond))
}

type contentOcurrence struct {
	Hash  string   `json:"cid"`
	Files []string `json:"paths"`
}
