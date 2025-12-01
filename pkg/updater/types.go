package updater

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/deatil/go-cryptobin/cryptobin/crypto"
	"github.com/tidwall/pretty"
)

type CryptoConfig struct {
	ProtectedKey       string `json:"protectedKey"`
	Version            string `json:"version"`
	NegotiationVersion string `json:"negotiationVersion"`
}

type RequestBody struct {
	Cipher string `json:"cipher"`
	Iv     string `json:"iv"`
}

type ResponseResult struct {
	ResponseCode       int    `json:"responseCode"`
	ErrMsg             string `json:"errMsg"`
	Body               any    `json:"body"`
	DecryptedBodyBytes []byte
}

func (r *ResponseResult) DecryptBody(key []byte) error {
	var m map[string]interface{}
	if r.Body == nil {
		return nil
	}
	bodyStr, ok := r.Body.(string)
	if !ok {
		return fmt.Errorf("response body is not a string")
	}
	if err := json.Unmarshal([]byte(bodyStr), &m); err != nil {
		return err
	}

	ivStr, ok := m["iv"].(string)
	if !ok {
		return fmt.Errorf("response missing 'iv' field or wrong type")
	}
	cipherStr, ok := m["cipher"].(string)
	if !ok {
		return fmt.Errorf("response missing 'cipher' field or wrong type")
	}
	iv, err := base64.StdEncoding.DecodeString(ivStr)
	if err != nil {
		return err
	}
	cipherBytes := crypto.FromBase64String(cipherStr).
		Aes().CTR().NoPadding().
		WithKey(key).WithIv(iv).
		Decrypt().
		ToBytes()

	r.DecryptedBodyBytes = cipherBytes
	return nil
}

func (r *ResponseResult) PrettyPrint() {
	var body map[string]interface{}
	if err := json.Unmarshal(r.DecryptedBodyBytes, &body); err == nil {
		m := map[string]interface{}{
			"responseCode": r.ResponseCode,
			"errMsg":       r.ErrMsg,
			"body":         body,
		}
		if bytes, err := json.Marshal(m); err == nil {
			fmt.Println(string(pretty.Color(pretty.Pretty(bytes), nil)))
		}
		return
	}

	m := map[string]interface{}{
		"responseCode": r.ResponseCode,
		"errMsg":       r.ErrMsg,
		"body":         string(r.DecryptedBodyBytes),
	}
	if bytes, err := json.Marshal(m); err == nil {
		fmt.Println(string(pretty.Color(pretty.Pretty(bytes), nil)))
	}
}

func (r *ResponseResult) AsJSON() []byte {
	var body map[string]interface{}
	if err := json.Unmarshal(r.DecryptedBodyBytes, &body); err == nil {
		m := map[string]interface{}{
			"responseCode": r.ResponseCode,
			"errMsg":       r.ErrMsg,
			"body":         body,
		}
		b, _ := json.Marshal(m)
		return b
	}
	m := map[string]interface{}{
		"responseCode": r.ResponseCode,
		"errMsg":       r.ErrMsg,
		"body":         string(r.DecryptedBodyBytes),
	}
	b, _ := json.Marshal(m)
	return b
}
