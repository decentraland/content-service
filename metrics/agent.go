package metrics

import (
	"fmt"
	"github.com/decentraland/content-service/config"
	"github.com/newrelic/go-agent"
	log "github.com/sirupsen/logrus"
	"gopkg.in/segmentio/analytics-go.v3"
	"net/http"
	"time"
)

type newrelicAgent interface {
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
	EndpointMetrics(tx string, w http.ResponseWriter, r *http.Request) Transaction
	RecordGetParcelMetadata(t time.Duration)
	RecordGetParcelContent(t time.Duration)
	RecordStoreContent(t time.Duration)
	RecordStoreMetadata(t time.Duration)
	RecordDCLAPIError(status int)
}

type segmentClient interface {
	RecordUpload(uploadId string, uploader string, parcels []string, files map[string][]string)
}

type Agent struct {
	newrelicAgent
	segmentClient
}

type newrelicAgentImpl struct {
	app newrelic.Application
}

func (a *newrelicAgentImpl) RecordBytesStored(fileSize int64) {
	if err := a.app.RecordCustomMetric("FileStored[bytes]", float64(fileSize)); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgentImpl) RecordBytesRetrieved(fileSize int64) {
	if err := a.app.RecordCustomMetric("FileRetrieved[bytes]", float64(fileSize)); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgentImpl) RecordUploadRequestValidationTime(t time.Duration) {
	if err := a.app.RecordCustomMetric("UploadValidationTime[msec|call]", toMillis(t)); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgentImpl) RecordRetrieveTime(t time.Duration) {
	if err := a.app.RecordCustomMetric("StorageDownloadTime[msec|call]", toMillis(t)); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgentImpl) RecordStorageTime(t time.Duration) {
	if err := a.app.RecordCustomMetric("StorageTime[msec|call]", toMillis(t)); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgentImpl) RecordDCLResponseTime(t time.Duration) {
	if err := a.app.RecordCustomMetric("DCLResponseTime[msec|call]", toMillis(t)); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgentImpl) RecordUploadRequestFiles(files int) {
	if err := a.app.RecordCustomMetric("UploadRequestFiles[files]", float64(files)); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgentImpl) RecordManifestSize(size int) {
	if err := a.app.RecordCustomMetric("Manifest[entries|call]", float64(size)); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgentImpl) RecordUploadProcessTime(t time.Duration) {
	if err := a.app.RecordCustomMetric("UploadProcessTime[msec|call]", toMillis(t)); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgentImpl) RecordUploadRequestParseTime(t time.Duration) {
	if err := a.app.RecordCustomMetric("UploadParseTime[msec|call]", toMillis(t)); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgentImpl) EndpointMetrics(tx string, w http.ResponseWriter, r *http.Request) Transaction {
	return &newRelicTx{a.app.StartTransaction(tx, w, r)}
}

func (a *newrelicAgentImpl) RecordUploadReqSize(size int) {
	if err := a.app.RecordCustomMetric("UploadRequestSize[bytes]", float64(size)); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgentImpl) RecordIsMemberTime(t time.Duration) {
	if err := a.app.RecordCustomMetric("IsMemberDuration[msec|call]", toMillis(t)); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgentImpl) RecordGetParcelMetadata(t time.Duration) {
	if err := a.app.RecordCustomMetric("GetParcelMetadata[msec|call]", toMillis(t)); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgentImpl) RecordGetParcelContent(t time.Duration) {
	if err := a.app.RecordCustomMetric("GetParcelContent[msec|call]", toMillis(t)); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgentImpl) RecordStoreContent(t time.Duration) {
	if err := a.app.RecordCustomMetric("StoreContent[msec|call]", toMillis(t)); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgentImpl) RecordStoreMetadata(t time.Duration) {
	if err := a.app.RecordCustomMetric("StoreMetadata[msec|call]", toMillis(t)); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgentImpl) RecordDCLAPIError(status int) {
	if err := a.app.RecordCustomMetric(fmt.Sprintf("DecentralandAPIError%d", status), float64(1)); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

type newrelicDummy struct{}

func (d *newrelicDummy) RecordBytesStored(fileSize int64)                  {}
func (d *newrelicDummy) RecordBytesRetrieved(fileSize int64)               {}
func (d *newrelicDummy) RecordUploadRequestValidationTime(t time.Duration) {}
func (d *newrelicDummy) RecordRetrieveTime(t time.Duration)                {}
func (d *newrelicDummy) RecordUploadReqSize(size int)                      {}
func (d *newrelicDummy) RecordDCLResponseTime(t time.Duration)             {}
func (d *newrelicDummy) RecordUploadRequestFiles(files int)                {}
func (d *newrelicDummy) RecordManifestSize(size int)                       {}
func (d *newrelicDummy) RecordUploadProcessTime(t time.Duration)           {}
func (d *newrelicDummy) RecordUploadRequestParseTime(t time.Duration)      {}
func (d *newrelicDummy) RecordIsMemberTime(t time.Duration)                {}
func (d *newrelicDummy) RecordStorageTime(t time.Duration)                 {}
func (d *newrelicDummy) RecordGetParcelMetadata(t time.Duration)           {}
func (d *newrelicDummy) RecordGetParcelContent(t time.Duration)            {}
func (d *newrelicDummy) RecordStoreContent(t time.Duration)                {}
func (d *newrelicDummy) RecordStoreMetadata(t time.Duration)               {}
func (d *newrelicDummy) RecordDCLAPIError(status int)                      {}
func (d *newrelicDummy) EndpointMetrics(tx string, w http.ResponseWriter, r *http.Request) Transaction {
	return &dummyTx{}
}

type Transaction interface {
	Close()
	ReportError(err error)
}

type newRelicTx struct {
	tx newrelic.Transaction
}

func (t *newRelicTx) Close() {
	if err := t.tx.End(); err != nil {
		log.Errorf("Failed to close New Relic Transaction: %s", err.Error())
	}
}
func (t *newRelicTx) ReportError(err error) {
	if err := t.tx.NoticeError(err); err != nil {
		log.Errorf("Failed to Report error: %s", err.Error())
	}
}

type dummyTx struct{}

func (t *dummyTx) Close()                {}
func (t *dummyTx) ReportError(err error) {}

type segmentClientImpl struct {
	client analytics.Client
}

func (sa *segmentClientImpl) RecordUpload(uploadId string, uploader string, parcels []string, files map[string][]string) {
	err := sa.client.Enqueue(analytics.Track{
		UserId: uploader,
		Event:  "Content Upload",
		Properties: analytics.NewProperties().
			Set("files", files).
			Set("parcels", parcels).
			Set("cid", uploadId),
	})
	if err != nil {
		log.Errorf("[SEGMENT] Failed to queue event : %s", err.Error())
	}
}

type segmentDummy struct {
	client analytics.Client
}

func (sa *segmentDummy) RecordUpload(uploadId string, uploader string, parcels []string, files map[string][]string) {
}

func Make(metrics config.Metrics) (*Agent, error) {
	nrAgent, err := buildNewRelicAgent(metrics.AppName, metrics.AppKey)
	if err != nil {
		return nil, err
	}

	segAgent, err := buildSegmentAgent(metrics.AnalyticsKey)
	if err != nil {
		return nil, err
	}

	return &Agent{nrAgent, segAgent}, nil
}

func buildNewRelicAgent(appName string, newrelicApiKey string) (newrelicAgent, error) {
	log.Debug("Initializing Newrelic Agent")
	if newrelicApiKey == "" {
		log.Debug("No Newrelic configuration found")
		return &newrelicDummy{}, nil
	}
	conf := newrelic.NewConfig(appName, newrelicApiKey)
	app, err := newrelic.NewApplication(conf)

	if err != nil {
		log.Errorf("Failed to initialize Newrelic agent: %s", err.Error())
		return nil, err
	}

	log.Infof("Newrelic agent loaded for: %s", appName)

	return &newrelicAgentImpl{app: app}, nil
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
