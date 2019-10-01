package integration

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/decentraland/content-service/data"
	"github.com/decentraland/content-service/internal/deployment"
	"github.com/decentraland/content-service/internal/entities"
	"github.com/decentraland/content-service/internal/ipfs"
	"github.com/decentraland/content-service/internal/routes"
	"github.com/decentraland/content-service/internal/storage"
	"github.com/decentraland/content-service/metrics"
	"github.com/decentraland/content-service/utils"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	iCore "github.com/ipsn/go-ipfs/core"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ginlogrus "github.com/toorop/gin-logrus"
)

type testRouter struct {
	*gin.Engine
	T *testing.T
}

type contentMetadata struct {
	Cid  string `json:"cid" validate:"required"`
	Name string `json:"name" validate:"required"`
}

type content struct {
	Metadata *contentMetadata
	data     string
}

func prepareEngine(t *testing.T, h *ipfs.IpfsHelper) testRouter {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)

	r := gin.New()
	r.Use(ginlogrus.Logger(l), gin.Recovery())
	gin.SetMode(gin.ReleaseMode)

	a, _ := metrics.Make(metrics.Config{
		AnalyticsKey: "",
		Enabled:      false,
		AppName:      "",
	})

	routes.AddRoutes(r, &routes.Config{
		Storage: storage.NewStorage(storage.ContentBucket{
			Bucket: "local-content",
			ACL:    "public-read",
			URL:    "http://localhost:4572/local-content",
		}, a),
		Ipfs:             h,
		Agent:            a,
		Log:              l,
		DclClient:        &mockDcl{},
		RpcClient:        &mockRpc{},
		Filter:           utils.NewContentTypeFilter([]string{".*"}),
		ParcelSizeLimit:  10000000,
		ParcelAssetLimit: 10000000,
		RequestTTL:       6000,
		MRepo: deployment.NewRepository(&deployment.Config{
			Bucket: "local-mappings",
			ACL:    "public-read",
			URL:    "http://localhost:4572/local-mappings",
		}),
	})
	return testRouter{Engine: r, T: t}
}

func TestBasicUpload(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	node, err := iCore.NewNode(ctx, nil)
	require.NoError(t, err)
	helper := &ipfs.IpfsHelper{node}

	r := prepareEngine(t, helper)

	pk, err := crypto.GenerateKey()
	require.NoError(t, err)

	address := getAddressFromKey(pk.PublicKey)

	positions := []string{"0,0"}
	sceneJson := generateSceneJson(address, positions, r.T)

	sCID, err := helper.CalculateCID(strings.NewReader(sceneJson))
	require.NoError(t, err)

	file := fmt.Sprintf(`{"something" : true, "random" : "%s"}`, uuid.New().String())
	cCID, err := helper.CalculateCID(strings.NewReader(file))
	require.NoError(t, err)

	required := []entities.ContentMapping{{Cid: sCID, Name: "scene.json"}}

	m := append(required, entities.ContentMapping{Cid: cCID, Name: "file.json"})
	mSlice, err := json.Marshal(m)
	require.NoError(t, err)
	mappings := string(mSlice)

	mCID, err := helper.CalculateCID(strings.NewReader(mappings))
	require.NoError(t, err)

	d := entities.Deploy{
		Required:  required,
		Positions: positions,
		Mappings:  mCID,
		Timestamp: time.Now().Unix(),
	}

	dSlice, err := json.Marshal(d)
	require.NoError(t, err)
	deploy := string(dSlice)

	dCID, err := helper.CalculateCID(strings.NewReader(deploy))
	require.NoError(t, err)

	now := time.Now().Unix()
	signature := signMessage(fmt.Sprintf("%s.%d", dCID, now), pk, r.T)

	p := entities.DeployProof{
		Signature: signature,
		Address:   address,
		ID:        dCID,
		Timestamp: now,
	}

	pSlice, err := json.Marshal(p)
	require.NoError(t, err)
	proof := string(pSlice)

	content := []content{
		{
			Metadata: &contentMetadata{
				Cid:  sCID,
				Name: "scene.json",
			},
			data: sceneJson,
		},
		{
			Metadata: &contentMetadata{
				Cid:  cCID,
				Name: "randomFile.json",
			},
			data: file,
		},
		{
			Metadata: &contentMetadata{
				Cid:  "mapping.json",
				Name: "mapping.json",
			},
			data: mappings,
		},
		{
			Metadata: &contentMetadata{
				Cid:  "deploy.json",
				Name: "deploy.json",
			},
			data: deploy,
		},
		{
			Metadata: &contentMetadata{
				Cid:  "proof.json",
				Name: "proof.json",
			},
			data: proof,
		},
	}

	r.UploadContent(content, http.StatusOK)
}

type mockRpc struct{}

func (r *mockRpc) ValidateDapperSignature(address, value, signature string) (bool, error) {
	return true, nil
}

type mockDcl struct{}

func (d *mockDcl) GetParcelAccessData(address string, x int64, y int64) (*data.AccessData, error) {
	return &data.AccessData{
		IsUpdateAuthorized: true,
	}, nil
}

func (tr testRouter) UploadContent(files []content, expectedStatus int) string {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	tr.loadContent(files, writer)

	err := writer.Close()
	require.NoError(tr.T, err)

	req, err := http.NewRequest("POST", "/api/v1/contents", body)
	require.NoError(tr.T, err)

	req.Header.Set("Content-Type", writer.FormDataContentType())

	code, response := tr.runRequest(req, expectedStatus)
	assert.Equal(tr.T, code, expectedStatus)
	if code != http.StatusOK {
		return response["error"].(string)
	}
	return ""
}

func (t testRouter) loadContent(files []content, w *multipart.Writer) {
	for _, file := range files {
		part, err := w.CreateFormFile(file.Metadata.Cid, file.Metadata.Name)
		require.NoError(t.T, err)
		_, err = io.Copy(part, strings.NewReader(file.data))
		require.NoError(t.T, err)
	}
}

func (tr testRouter) runRequest(req *http.Request, expectedStatus int) (int, map[string]interface{}) {
	w := httptest.NewRecorder()
	tr.ServeHTTP(w, req)
	require.Equal(tr.T, expectedStatus, w.Code)

	var response map[string]interface{}
	bBytes := w.Body.Bytes()
	if bBytes != nil {
		err := json.Unmarshal(bBytes, &response)
		require.NoError(tr.T, err)
	}
	return w.Code, response
}

func generateSceneJson(ownerAddress string, parcels []string, t *testing.T) string {
	read, err := ioutil.ReadFile("../resources/scene-template.json")
	require.NoError(t, err)
	out := strings.Replace(string(read), "${PARCELS}", strings.Join(parcels, `","`), -1)
	out = strings.Replace(string(out), "${OWNER}", ownerAddress, -1)
	out = strings.Replace(string(out), "${BASE_PARCEL}", parcels[0], -1)

	return out
}

func getAddressFromKey(pk ecdsa.PublicKey) string {
	a := crypto.PubkeyToAddress(pk)
	add := a.String()
	return add
}

func signMessage(message string, key *ecdsa.PrivateKey, t *testing.T) string {
	mBytes, _ := core.SignHash([]byte(message))

	sigBytes, err := crypto.Sign(mBytes, key)
	require.NoError(t, err)

	signature := hexutil.Encode(sigBytes)

	return signature
}
