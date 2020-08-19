package rest

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"
)

var defaultClientTimeout = 90 * time.Second

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

var client = &http.Client{
	// Notifications need be hold request 60 second，so make sure client is greater than 60 second。
	Timeout: defaultClientTimeout,
}

func Do(method, url string, headers map[string]string, v interface{}) (status int, err error) {
	var req *http.Request
	req, err = http.NewRequest(method, url, nil)
	if err != nil {
		return
	}

	for key, val := range headers {
		req.Header.Set(key, val)
	}

	var body []byte
	status, body, err = parseResponseBody(client, req)
	if err != nil {
		return
	}

	if status == http.StatusOK {
		err = json.Unmarshal(body, v)
	}
	return
}

func parseResponseBody(doer Doer, req *http.Request) (int, []byte, error) {
	resp, err := doer.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	return resp.StatusCode, body, nil
}
