package metrics

import (
	"github.com/newrelic/go-agent"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type Agent interface {
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
}

type newrelicAgent struct {
	app newrelic.Application
}

func (a *newrelicAgent) RecordBytesStored(fileSize int64) {
	if err := a.app.RecordCustomMetric("FileStored[bytes]", float64(fileSize)); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgent) RecordBytesRetrieved(fileSize int64) {
	if err := a.app.RecordCustomMetric("FileRetrieved[bytes]", float64(fileSize)); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgent) RecordUploadRequestValidationTime(t time.Duration) {
	if err := a.app.RecordCustomMetric("UploadValidationTime[msec|call]", toMillis(t.Nanoseconds())); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgent) RecordRetrieveTime(t time.Duration) {
	if err := a.app.RecordCustomMetric("StorageDownloadTime[msec|call]", toMillis(t.Nanoseconds())); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgent) RecordStorageTime(t time.Duration) {
	if err := a.app.RecordCustomMetric("StorageTime[msec|call]", toMillis(t.Nanoseconds())); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgent) RecordDCLResponseTime(t time.Duration) {
	if err := a.app.RecordCustomMetric("DCLResponseTime[msec|call]", toMillis(t.Nanoseconds())); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgent) RecordUploadRequestFiles(files int) {
	if err := a.app.RecordCustomMetric("UloadRequestFiles[files]", float64(files)); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgent) RecordManifestSize(size int) {
	if err := a.app.RecordCustomMetric("Manifest[entries|call]", float64(size)); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgent) RecordUploadProcessTime(t time.Duration) {
	if err := a.app.RecordCustomMetric("UploadProcessTime[msec|call]", toMillis(t.Nanoseconds())); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgent) RecordUploadRequestParseTime(t time.Duration) {
	if err := a.app.RecordCustomMetric("UploadParseTime[msec|call]", toMillis(t.Nanoseconds())); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgent) EndpointMetrics(tx string, w http.ResponseWriter, r *http.Request) Transaction {
	return &newRelicTx{a.app.StartTransaction(tx, w, r)}
}

func (a *newrelicAgent) RecordUploadReqSize(size int) {
	if err := a.app.RecordCustomMetric("UploadRequestSize[bytes]", float64(size)); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

func (a *newrelicAgent) RecordIsMemberTime(t time.Duration) {
	if err := a.app.RecordCustomMetric("IsMemberDuration[msec|call]", toMillis(t.Nanoseconds())); err != nil {
		log.Errorf("Metrics agent failed: %s", err.Error())
	}
}

type dummy struct{}

func (d *dummy) RecordBytesStored(fileSize int64)                  {}
func (d *dummy) RecordBytesRetrieved(fileSize int64)               {}
func (d *dummy) RecordUploadRequestValidationTime(t time.Duration) {}
func (d *dummy) RecordRetrieveTime(t time.Duration)                {}
func (d *dummy) RecordUploadReqSize(size int)                      {}
func (d *dummy) RecordDCLResponseTime(t time.Duration)             {}
func (d *dummy) RecordUploadRequestFiles(files int)                {}
func (d *dummy) RecordManifestSize(size int)                       {}
func (d *dummy) RecordUploadProcessTime(t time.Duration)           {}
func (d *dummy) RecordUploadRequestParseTime(t time.Duration)      {}
func (d *dummy) RecordIsMemberTime(t time.Duration)                {}
func (d *dummy) RecordStorageTime(t time.Duration)                 {}
func (d *dummy) EndpointMetrics(tx string, w http.ResponseWriter, r *http.Request) Transaction {
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

func Make(appName string, newrelicApiKey string) (Agent, error) {
	log.Debug("Initializing Metrics Agent")
	if newrelicApiKey == "" {
		log.Debug("No Agent configuration found")
		return &dummy{}, nil
	}

	config := newrelic.NewConfig(appName, newrelicApiKey)
	app, err := newrelic.NewApplication(config)

	if err != nil {
		log.Errorf("Failed to initialize metrics agent: %s", err.Error())
		return nil, err
	}

	log.Infof("Newrelic agent loaded for: %s", appName)

	return &newrelicAgent{app: app}, nil
}

func toMillis(duration int64) float64 {
	return float64(duration / int64(time.Millisecond))
}
