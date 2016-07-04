// Tests the fetcher package.
package fetcher

import (
	"bytes"
	"expvar"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

var (
	fakeResponse = &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewBufferString("fake fetch")),
	}
)

func TestNewFileFetch_PassNoCookies(t *testing.T) {
	actual, err := NewFileFetch("http://test.only-testing.com/wheee", os.TempDir(), nil)
	assert.NoError(t, err, "Errors returned creating a new FileFetch object.")
	assert.NotEmpty(t, actual, "Empty FileFetch returned.")
	assert.NotEmpty(t, actual.HttpClient.Jar, "Jar not set correctly.")
}

func TestNewFileFetch_FailBadUrl(t *testing.T) {
	a, e := NewFileFetch("bad\test/url%%", os.TempDir(), nil)
	assert.Error(t, e, "Test url should not have worked.")
	assert.Nil(t, a, "FileFetch object returned, unexpected.")
}

func TestRetrieveEpisode(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	testUrl := "http://this.is-test.net/foobar"
	httpmock.RegisterResponder("GET", testUrl, func(req *http.Request) (*http.Response, error) {
		return fakeResponse, nil
	})

	url, _ := url.Parse(testUrl)
	fakeFF := &FileFetch{
		HttpClient:   &http.Client{},
		Url:          url,
		SaveLocation: filepath.Join(os.TempDir(), "test.html"),
	}
	err := fakeFF.RetrieveEpisode()
	assert.NoError(t, err, "Error in fake fetch: %s", err)
}

func TestUpdateResultMap(t *testing.T) {
	fetchResultMap = expvar.NewMap("test_fetch_results").Init()
	updateResultMap("ryan")
	updateResultMap("daniel")
	updateResultMap("ryan")
	updateResultMap("monkeys")
	fetchResultMap.Do(func(kv expvar.KeyValue) {
		if kv.Key == "ryan" {
			assert.Equal(t, kv.Value.String(), "2")
		} else if kv.Key == "daniel" {
			assert.Equal(t, kv.Value.String(), "1")
		} else if kv.Key == "monkeys" {
			assert.Equal(t, kv.Value.String(), "1")
		} else {
			t.Fatal("A value in the result map is messing things up.")
		}
	})
}
