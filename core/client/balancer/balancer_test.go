package balancer

import (
	"sync"
	"testing"
	"time"

	"gopkg.in/go-playground/assert.v1"

	"github.com/sixgoatsh/agollo/core/client"
	"github.com/sixgoatsh/agollo/core/mock"
	"github.com/sixgoatsh/agollo/pkg/log"
)

func TestAutoFetchBalancer(t *testing.T) {
	refreshIntervalInSecond := time.Second * 2

	expected := []client.ConfigServerResp{
		{
			AppName:     "APOLLO-CONFIGSERVICE",
			InstanceID:  "localhost:apollo-configservice:8080",
			HomePageURL: "http://127.0.0.1:8080",
		},
	}

	var wg sync.WaitGroup
	go func() {
		<-time.After(refreshIntervalInSecond)

		expected = append(expected, client.ConfigServerResp{
			AppName:     "APOLLO-CONFIGSERVICE",
			InstanceID:  "localhost:apollo-configservice:8081",
			HomePageURL: "http://127.0.0.1:8081",
		})

		wg.Done()
	}()

	metaServerClient := &mock.MetaServerClient{
		ConfigServers: func(metaServerURL, appID string) (int, []client.ConfigServerResp, error) {
			return 200, expected, nil
		},
	}

	b, err := NewAutoFetchBalancer("", "", metaServerClient, refreshIntervalInSecond, log.NewLogger())
	if err != nil {
		t.Fatal(err)
	}

	actual, err := b.Select()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, expected[0].HomePageURL, actual)

	wg.Wait()

	for i := 0; i < 10; i++ {
		actual, err := b.Select()
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, expected[i%len(expected)].HomePageURL, actual)
	}
}

func TestRoundRobin(t *testing.T) {
	expected := []string{
		"http://127.0.0.1:8080/",
		"http://127.0.0.1:8081/",
	}

	lb := NewRoundRobin(expected)

	for i := 0; i < 10; i++ {
		actual, err := lb.Select()
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, expected[i%len(expected)], actual)
	}

}
