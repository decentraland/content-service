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

	"github.com/decentraland/content-service/internal/decentraland"

	"github.com/decentraland/content-service/internal/deployment"
	"github.com/decentraland/content-service/internal/entities"
	"github.com/decentraland/content-service/internal/ipfs"
	"github.com/decentraland/content-service/internal/metrics"
	"github.com/decentraland/content-service/internal/routes"
	"github.com/decentraland/content-service/internal/storage"
	"github.com/decentraland/content-service/internal/utils"
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

type uploadContent struct {
	c []content
}

func newUploadContent() uploadContent {
	return uploadContent{
		c: []content{},
	}
}

func (uc uploadContent) addContent(data string, cid string, name string) uploadContent {
	uc.c = append(uc.c, content{
		Metadata: &contentMetadata{
			Cid:  cid,
			Name: name,
		},
		data: data,
	})
	return uc
}

func (uc uploadContent) append(c content) uploadContent {
	uc.c = append(uc.c, c)
	return uc
}

func prepareEngine(t *testing.T, h *ipfs.IpfsHelper) testRouter {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)

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
	sceneJson, sCID := generateSceneJson(address, positions, r.T, helper)

	file, fCID := generateRandomFile(r.T, helper)
	require.NoError(t, err)

	required := []entities.ContentMapping{{Cid: sCID, Name: "scene.json"}}

	mappings, mCID := generateContentData(append(required, entities.ContentMapping{Cid: fCID, Name: "file.json"}), r.T, helper)

	deploy, dCID := generateDeployJson(required, positions, mCID, time.Now().Unix(), r.T, helper)

	now := time.Now().Unix()
	signature := signMessage(fmt.Sprintf("%s.%d", dCID, now), pk, r.T)
	proof, _ := generateDeployProof(signature, address, dCID, time.Now().Unix(), r.T, helper)

	content := newUploadContent().
		addContent(sceneJson, sCID, "scene.json").
		addContent(file, fCID, "file.json").
		addContent(mappings, "mapping.json", "mapping.json").
		addContent(deploy, "deploy.json", "deploy.json").
		addContent(proof, "proof.json", "proof.json")

	r.UploadContent(content.c, http.StatusOK)
}

func TestMissingRequiredFiles(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	node, err := iCore.NewNode(ctx, nil)
	require.NoError(t, err)
	helper := &ipfs.IpfsHelper{node}

	r := prepareEngine(t, helper)

	pk, err := crypto.GenerateKey()
	require.NoError(t, err)

	address := getAddressFromKey(pk.PublicKey)

	positions := []string{"0,0"}

	file, fCID := generateRandomFile(r.T, helper)
	require.NoError(t, err)

	required := []entities.ContentMapping{{Cid: uuid.New().String(), Name: "scene.json"}}

	mappings, mCID := generateContentData(append(required, entities.ContentMapping{Cid: fCID, Name: "file.json"}), r.T, helper)

	deploy, dCID := generateDeployJson(required, positions, mCID, time.Now().Unix(), r.T, helper)

	now := time.Now().Unix()
	signature := signMessage(fmt.Sprintf("%s.%d", dCID, now), pk, r.T)
	proof, _ := generateDeployProof(signature, address, dCID, time.Now().Unix(), r.T, helper)

	content := newUploadContent().
		addContent(file, fCID, "randomFile.json").
		addContent(mappings, "mapping.json", "mapping.json").
		addContent(deploy, "deploy.json", "deploy.json").
		addContent(proof, "proof.json", "proof.json")

	errMsg := r.UploadContent(content.c, http.StatusBadRequest)
	assert.Equal(r.T, "missing required files", errMsg)
}

func TestMissingFile(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	node, err := iCore.NewNode(ctx, nil)
	require.NoError(t, err)
	helper := &ipfs.IpfsHelper{node}

	r := prepareEngine(t, helper)

	pk, err := crypto.GenerateKey()
	require.NoError(t, err)

	address := getAddressFromKey(pk.PublicKey)

	positions := []string{"0,0"}
	sceneJson, sCID := generateSceneJson(address, positions, r.T, helper)

	_, fCID := generateRandomFile(r.T, helper)
	require.NoError(t, err)

	required := []entities.ContentMapping{{Cid: sCID, Name: "scene.json"}}

	mappings, mCID := generateContentData(append(required, entities.ContentMapping{Cid: fCID, Name: "file.json"}), r.T, helper)

	deploy, dCID := generateDeployJson(required, positions, mCID, time.Now().Unix(), r.T, helper)

	now := time.Now().Unix()
	signature := signMessage(fmt.Sprintf("%s.%d", dCID, now), pk, r.T)
	proof, _ := generateDeployProof(signature, address, dCID, time.Now().Unix(), r.T, helper)

	// The request will not contain the 'file.json'
	content := newUploadContent().
		addContent(sceneJson, sCID, "scene.json").
		addContent(mappings, "mapping.json", "mapping.json").
		addContent(deploy, "deploy.json", "deploy.json").
		addContent(proof, "proof.json", "proof.json")

	errMsg := r.UploadContent(content.c, http.StatusBadRequest)
	assert.Equal(r.T, fmt.Sprintf("file: %s not found", fCID), errMsg)
}

func TestPartialUpload(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	node, err := iCore.NewNode(ctx, nil)
	require.NoError(t, err)
	helper := &ipfs.IpfsHelper{node}

	r := prepareEngine(t, helper)

	pk, err := crypto.GenerateKey()
	require.NoError(t, err)

	address := getAddressFromKey(pk.PublicKey)

	positions := []string{"0,0"}
	sceneJson, sCID := generateSceneJson(address, positions, r.T, helper)

	file, fCID := generateRandomFile(r.T, helper)
	require.NoError(t, err)

	required := []entities.ContentMapping{{Cid: sCID, Name: "scene.json"}}

	mappings, mCID := generateContentData(append(required, entities.ContentMapping{Cid: fCID, Name: "file.json"}), r.T, helper)

	deploy, dCID := generateDeployJson(required, positions, mCID, time.Now().Unix(), r.T, helper)

	now := time.Now().Unix()
	signature := signMessage(fmt.Sprintf("%s.%d", dCID, now), pk, r.T)
	proof, _ := generateDeployProof(signature, address, dCID, time.Now().Unix(), r.T, helper)

	// First time i need to upload all files
	content := newUploadContent().
		addContent(sceneJson, sCID, "scene.json").
		addContent(file, fCID, "file.json").
		addContent(mappings, "mapping.json", "mapping.json").
		addContent(deploy, "deploy.json", "deploy.json").
		addContent(proof, "proof.json", "proof.json")

	r.UploadContent(content.c, http.StatusOK)

	// Second time i can avoid uploading non required files that had been previously uploade
	content = newUploadContent().
		addContent(sceneJson, sCID, "scene.json").
		addContent(mappings, "mapping.json", "mapping.json").
		addContent(deploy, "deploy.json", "deploy.json").
		addContent(proof, "proof.json", "proof.json")

	r.UploadContent(content.c, http.StatusOK)
}

func TestNotMatchingSignature(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	node, err := iCore.NewNode(ctx, nil)
	require.NoError(t, err)
	helper := &ipfs.IpfsHelper{node}

	r := prepareEngine(t, helper)

	pk, err := crypto.GenerateKey()
	require.NoError(t, err)

	address := getAddressFromKey(pk.PublicKey)

	positions := []string{"0,0"}
	sceneJson, sCID := generateSceneJson(address, positions, r.T, helper)
	required := []entities.ContentMapping{{Cid: sCID, Name: "scene.json"}}

	mappings, mCID := generateContentData(required, r.T, helper)

	deploy, dCID := generateDeployJson(required, positions, mCID, time.Now().Unix(), r.T, helper)

	alternativePk, err := crypto.GenerateKey()

	now := time.Now().Unix()
	signature := signMessage(fmt.Sprintf("%s.%d", dCID, now), alternativePk, r.T)
	proof, _ := generateDeployProof(signature, address, dCID, time.Now().Unix(), r.T, helper)

	content := newUploadContent().
		addContent(sceneJson, sCID, "scene.json").
		addContent(mappings, "mapping.json", "mapping.json").
		addContent(deploy, "deploy.json", "deploy.json").
		addContent(proof, "proof.json", "proof.json")

	errMsg := r.UploadContent(content.c, http.StatusBadRequest)
	assert.Equal(r.T, "Signature is invalid", errMsg)
}

func TestContentStatus(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	node, err := iCore.NewNode(ctx, nil)
	require.NoError(t, err)
	helper := &ipfs.IpfsHelper{node}

	r := prepareEngine(t, helper)

	pk, err := crypto.GenerateKey()
	require.NoError(t, err)

	address := getAddressFromKey(pk.PublicKey)

	positions := []string{"0,0"}
	sceneJson, sCID := generateSceneJson(address, positions, r.T, helper)

	file, fCID := generateRandomFile(r.T, helper)
	require.NoError(t, err)

	required := []entities.ContentMapping{{Cid: sCID, Name: "scene.json"}}

	mappings, mCID := generateContentData(append(required, entities.ContentMapping{Cid: fCID, Name: "file.json"}), r.T, helper)

	deploy, dCID := generateDeployJson(required, positions, mCID, time.Now().Unix(), r.T, helper)

	now := time.Now().Unix()
	signature := signMessage(fmt.Sprintf("%s.%d", dCID, now), pk, r.T)
	proof, _ := generateDeployProof(signature, address, dCID, time.Now().Unix(), r.T, helper)

	// First time i need to upload all files
	content := newUploadContent().
		addContent(sceneJson, sCID, "scene.json").
		addContent(file, fCID, "file.json").
		addContent(mappings, "mapping.json", "mapping.json").
		addContent(deploy, "deploy.json", "deploy.json").
		addContent(proof, "proof.json", "proof.json")

	r.UploadContent(content.c, http.StatusOK)

	fakeId := uuid.New().String()
	resp := r.CheckContentStatus([]string{sCID, fCID, fakeId}, http.StatusOK)

	assert.True(r.T, resp[sCID].(bool))
	assert.True(r.T, resp[fCID].(bool))
	assert.False(r.T, resp[fakeId].(bool))
}

type mockRpc struct{}

func (r *mockRpc) ValidateDapperSignature(address, value, signature string) (bool, error) {
	return true, nil
}

type mockDcl struct{}

func (d *mockDcl) GetParcelAccessData(address string, x int64, y int64) (*decentraland.AccessData, error) {
	return &decentraland.AccessData{
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

func (tr testRouter) CheckContentStatus(cids []string, expectedStatus int) map[string]interface{} {

	body := fmt.Sprintf(`{ "content" : ["%s"] }`, strings.Join(cids, `","`))

	req, err := http.NewRequest("POST", "/api/v1/asset_status", strings.NewReader(body))
	require.NoError(tr.T, err)

	code, response := tr.runRequest(req, expectedStatus)
	assert.Equal(tr.T, code, expectedStatus)
	return response
}

func TestMissingDeploymentFiles(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	node, err := iCore.NewNode(ctx, nil)
	require.NoError(t, err)
	helper := &ipfs.IpfsHelper{node}

	r := prepareEngine(t, helper)

	pk, err := crypto.GenerateKey()
	require.NoError(t, err)

	address := getAddressFromKey(pk.PublicKey)

	positions := []string{"0,0"}
	sceneJson, sCID := generateSceneJson(address, positions, r.T, helper)

	required := []entities.ContentMapping{{Cid: sCID, Name: "scene.json"}}

	mappings, mCID := generateContentData(required, r.T, helper)

	deploy, dCID := generateDeployJson(required, positions, mCID, time.Now().Unix(), r.T, helper)

	now := time.Now().Unix()
	signature := signMessage(fmt.Sprintf("%s.%d", dCID, now), pk, r.T)
	proof, _ := generateDeployProof(signature, address, dCID, time.Now().Unix(), r.T, helper)

	t.Run("Missing deploy.json", func(t *testing.T) {
		content := newUploadContent().
			addContent(sceneJson, sCID, "scene.json").
			addContent(mappings, "mapping.json", "mapping.json").
			addContent(proof, "proof.json", "proof.json")

		r.UploadContent(content.c, http.StatusBadRequest)

	})

	t.Run("Missing mapping.json", func(t *testing.T) {
		content := newUploadContent().
			addContent(sceneJson, sCID, "scene.json").
			addContent(deploy, "deploy.json", "deploy.json").
			addContent(proof, "proof.json", "proof.json")

		r.UploadContent(content.c, http.StatusBadRequest)

	})

	t.Run("Missing proof.json", func(t *testing.T) {
		content := newUploadContent().
			addContent(sceneJson, sCID, "scene.json").
			addContent(mappings, "mapping.json", "mapping.json").
			addContent(deploy, "deploy.json", "deploy.json")

		r.UploadContent(content.c, http.StatusBadRequest)
	})
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

func generateSceneJson(ownerAddress string, parcels []string, t *testing.T, h *ipfs.IpfsHelper) (string, string) {
	read, err := ioutil.ReadFile("../resources/scene-template.json")
	require.NoError(t, err)
	out := strings.Replace(string(read), "${PARCELS}", strings.Join(parcels, `","`), -1)
	out = strings.Replace(string(out), "${OWNER}", ownerAddress, -1)
	out = strings.Replace(string(out), "${BASE_PARCEL}", parcels[0], -1)

	sCID, err := h.CalculateCID(strings.NewReader(out))
	require.NoError(t, err)

	return out, sCID
}

func generateRandomFile(t *testing.T, h *ipfs.IpfsHelper) (string, string) {
	file := fmt.Sprintf(`{"something" : true, "random" : "%s"}`, uuid.New().String())
	fCID, err := h.CalculateCID(strings.NewReader(file))
	require.NoError(t, err)
	return file, fCID
}

func generateDeployJson(r []entities.ContentMapping, pos []string, mHash string, ts int64,
	t *testing.T, h *ipfs.IpfsHelper) (string, string) {
	d := entities.Deploy{
		Required:  r,
		Positions: pos,
		Mappings:  mHash,
		Timestamp: ts,
	}
	return generateContentData(d, t, h)
}

func generateDeployProof(s string, a string, id string, ts int64, t *testing.T, h *ipfs.IpfsHelper) (string, string) {
	p := entities.DeployProof{
		Signature: s,
		Address:   a,
		ID:        id,
		Timestamp: ts,
	}
	return generateContentData(p, t, h)
}

func generateContentData(entity interface{}, t *testing.T, h *ipfs.IpfsHelper) (string, string) {
	slice, err := json.Marshal(entity)
	require.NoError(t, err)
	str := string(slice)

	cid, err := h.CalculateCID(strings.NewReader(str))
	require.NoError(t, err)

	return str, cid
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
