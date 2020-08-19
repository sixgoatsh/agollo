package auth

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"time"
)

const (
	AUTHORIZATION_FORMAT      = "Apollo %s:%s"
	DELIMITER                 = "\n"
	HTTP_HEADER_AUTHORIZATION = "Authorization"
	HTTP_HEADER_TIMESTAMP     = "Timestamp"
)

func signature(timestamp, url, accessKey string) string {
	stringToSign := timestamp + DELIMITER + url

	key := []byte(accessKey)
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func HttpHeader(accessKey string, appID, uri string) map[string]string {
	headers := map[string]string{}
	if "" == accessKey {
		return headers
	}

	timestamp := fmt.Sprintf("%v", time.Now().UnixNano()/int64(time.Millisecond))
	signature := signature(timestamp, uri, accessKey)

	headers[HTTP_HEADER_AUTHORIZATION] = fmt.Sprintf(AUTHORIZATION_FORMAT, appID, signature)
	headers[HTTP_HEADER_TIMESTAMP] = timestamp

	return headers
}
