package hikcentral

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"
)

// this proved a helpful reference when deciphering the docs on signing requests:
// https://github.com/zxbit2011/hikvisionOpenAPIGo/blob/main/sdk.go

func prepareReq(req *http.Request, body []byte, secret, key string) error {
	if len(body) > 0 {
		c, err := contentMd5(body)
		if err != nil {
			return err
		}
		req.Header.Set("Content-MD5", c)
	}
	req.Header.Set("X-Ca-Key", key)
	return signReq(req, secret)
}

func signReq(req *http.Request, secret string) error {
	str, headers := signatureString(req)
	req.Header.Set("X-Ca-Signature-Headers", headers)

	mac := hmac.New(sha256.New, []byte(secret))
	_, err := mac.Write([]byte(str))
	if err != nil {
		return err
	}
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	req.Header.Set("X-Ca-Signature", signature)

	return nil
}

// contentMd5 calculates the "Content-MD5" part of the signature string
// from the docs:
//
// The Content-MD5 is the value of digest calculated by MD5 algorithm in the request body and
// processed by BASE64 algorithm, and the body must be on non-form format. E.g., String content-
// MD5=Base64.encodeBase64(MD5(bodyStream.getbytes("UTF-8"))).
func contentMd5(body []byte) (string, error) {
	hash := md5.Sum(body)
	return base64.StdEncoding.EncodeToString(hash[:]), nil
}

// signatureString calculates the signature string + headers used for "AK/SK digest auth".
// from the docs:
//
// The signature string before calculating consists of HTTP method, headers, and URI, which is defined as below:
//
//	HTTP METHOD "\n"
//	Accept "\n"
//	Content-MD5 "\n"
//	Content-Type "\n"
//	Date "\n"
//	Headers
//	Uri
func signatureString(req *http.Request) (sigString, sigHeaders string) {
	var parts []string

	if req.Method != "" {
		parts = append(parts, strings.ToUpper(req.Method))
	} else {
		parts = append(parts, "GET")
	}
	parts = append(parts, "\n")
	headers := []string{
		"Accept",
		"Content-MD5",
		"Content-Type",
		"Date",
	}
	for i := 0; i < len(headers); i++ {
		value := req.Header.Get(headers[i])
		if value != "" {
			parts = append(parts, value, "\n")
		}
	}
	headers = []string{
		"X-Ca-Key",
		"X-Ca-Nonce",
		"X-Ca-Timestamp",
	}
	var includes []string
	for i := 0; i < len(headers); i++ {
		value := req.Header.Get(headers[i])
		if value != "" {
			header := strings.ToLower(headers[i])
			// <header>:<value>\n
			parts = append(parts, header, ":", value, "\n")
			includes = append(includes, header)
		}
	}

	parts = append(parts, req.URL.RequestURI())

	return strings.Join(parts, ""), strings.Join(includes, ",")
}
