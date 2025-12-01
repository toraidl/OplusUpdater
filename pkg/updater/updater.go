package updater

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/deatil/go-cryptobin/cryptobin/crypto"
	"github.com/go-resty/resty/v2"
)

var allowedReqModes = map[string]struct{}{
	"manual":      {},
	"server_auto": {},
	"client_auto": {},
	"taste":       {},
}

func validateReqMode(mode string) (string, error) {
	m := strings.TrimSpace(mode)
	if m == "" {
		return "manual", nil
	}
	if _, ok := allowedReqModes[m]; ok {
		return m, nil
	}
	return "", fmt.Errorf("invalid reqmode: %s", m)
}

type QueryUpdateArgs struct {
	OtaVersion string
	Region     string
	Model      string
	NvCarrier  string
	GUID       string
	Proxy      string
	Gray       bool
	Mode       string
}

func (args *QueryUpdateArgs) normalize() {
	if len(strings.Split(args.OtaVersion, "_")) < 3 || len(strings.Split(args.OtaVersion, ".")) < 3 {
		args.OtaVersion += ".01_0001_197001010000"
	}
	if r := strings.TrimSpace(args.Region); len(r) == 0 {
		args.Region = RegionCn
	}
	if m := strings.TrimSpace(args.Model); len(m) == 0 {
		args.Model = strings.Split(args.OtaVersion, "_")[0]
	}
}

func buildCryptoMaterials(config *Config) ([]byte, []byte, string, string, error) {
	iv, err := RandomIV()
	if err != nil {
		return nil, nil, "", "", err
	}
	key, err := RandomKey()
	if err != nil {
		return nil, nil, "", "", err
	}
	protectedKey, err := GenerateProtectedKey(key, []byte(config.PublicKey))
	if err != nil {
		return nil, nil, "", "", err
	}
	version := GenerateProtectedVersion()
	return iv, key, protectedKey, version, nil
}

func buildHeaders(config *Config, args *QueryUpdateArgs, deviceID string, reqMode string, protectedKeyHeader string) map[string]string {
	headers := map[string]string{
		"language":       config.Language,
		"androidVersion": "unknown",
		"colorOSVersion": "unknown",
		"romVersion":     "unknown",
		"otaVersion":     args.OtaVersion,
		"model":          args.Model,
		"mode":           reqMode,
		"nvCarrier":      args.NvCarrier,
		"infVersion":     "1",
		"version":        config.Version,
		"deviceId":       deviceID,
		"Content-Type":   "application/json; charset=utf-8",
	}
	headers["protectedKey"] = protectedKeyHeader
	return headers
}

func buildRequestBody(key, iv []byte, guid string) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"mode":     "0",
		"time":     time.Now().UnixMilli(),
		"isRooted": "0",
		"isLocked": true,
		"type":     "0",
		"deviceId": guid,
		"opex":     map[string]any{"check": true},
	})
	if err != nil {
		return "", err
	}
	body, err := json.Marshal(RequestBody{
		Cipher: crypto.FromBytes(payload).
			Aes().CTR().NoPadding().
			WithKey(key).WithIv(iv).
			Encrypt().
			ToBase64String(),
		Iv: base64.StdEncoding.EncodeToString(iv),
	})
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func doRequest(headers map[string]string, body string, host string, proxy string) (*resty.Response, error) {
	endpoint := url.URL{Host: host, Scheme: "https", Path: "/update/v5"}
	client := resty.New()
	if p := strings.TrimSpace(proxy); len(p) > 0 {
		client.SetProxy(p)
	}
	resp, err := client.R().
		SetHeaders(headers).
		SetBody(map[string]string{"params": body}).
		Post(endpoint.String())
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func QueryUpdate(args *QueryUpdateArgs) (*ResponseResult, error) {
	args.normalize()

	config := GetConfig(Region(args.Region), args.Gray)
	if args.NvCarrier == "" {
		args.NvCarrier = config.CarrierID
	}
	iv, key, protectedKey, version, err := buildCryptoMaterials(config)
	if err != nil {
		return nil, err
	}

	deviceID := GenerateDefaultDeviceID()
	GUID := GenerateDefaultDeviceID()
	if strings.TrimSpace(args.GUID) != "" {
		GUID = strings.ToLower(args.GUID)
	}

	reqMode, err := validateReqMode(args.Mode)
	if err != nil {
		return nil, err
	}

	pkm := map[string]CryptoConfig{
		"SCENE_1": {
			ProtectedKey:       protectedKey,
			Version:            version,
			NegotiationVersion: config.PublicKeyVersion,
		},
	}
	pk, err := json.Marshal(pkm)
	if err != nil {
		return nil, err
	}
	headers := buildHeaders(config, args, deviceID, reqMode, string(pk))

	requestBody, err := buildRequestBody(key, iv, GUID)
	if err != nil {
		return nil, err
	}

	response, err := doRequest(headers, requestBody, config.Host, args.Proxy)

	if err != nil {
		return nil, err
	}

	var responseResult *ResponseResult
	if unmarshalErr := json.Unmarshal(response.Body(), &responseResult); unmarshalErr != nil {
		return nil, unmarshalErr
	}

	if err := responseResult.DecryptBody(key); err != nil {
		return nil, err
	}

	return responseResult, nil
}
