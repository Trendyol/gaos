package test

import (
	"encoding/json"
	"fmt"
	"github.com/Trendyol/gaos/runner"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"io/ioutil"
	"net/http"
	"sync"
	"testing"
	"time"
)

type ApplicationError struct {
	Title         string                   `json:"title,omitempty"`
	Status        int                      `json:"status,omitempty"`
	Detail        string                   `json:"detail,omitempty"`
	Host          string                   `json:"host,omitempty"`
	RequestUri    string                   `json:"requestUri,omitempty"`
	RequestMethod string                   `json:"requestMethod,omitempty"`
	Instant       string                   `json:"instant,omitempty"`
	ErrorDetails  []ApplicationErrorDetail `json:"errorDetails,omitempty"`
	Cause         string                   `json:"cause,omitempty"`
	CorrelationID string                   `json:"correlationId,omitempty"`
}

type ApplicationErrorDetail struct {
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

type GaosTest struct {
	ApiUrl string
	ApiPort int32
	ApiHttpMethod string
	DeepBackendUrl string
	DeepBackendPort int32
	DeepBackendHttpMethod string
	DeepBackendCountCheckPath string
	MockContainerName string
	T *testing.T
}

func Test_RunGaosTestSuite(t *testing.T) {
	suite.Run(t, new(GaosTestSuite))
}

type GaosTestSuite struct {
	suite.Suite
}

func (suite *GaosTestSuite) SetupSuite() {

}

func (suite *GaosTestSuite) RunGaosTests() {
	t := suite.T()
	gaosTests, err := readGaosTestFile()

	assert.Nil(t, err)

	if err != nil {
		return
	}

	for _, gaosTest := range gaosTests {
		gaosTest.T = t
		gaosTest.circuitBreakerTest()
	}
}

func readGaosTestFile() ([]GaosTest, error) {
	var gaosTests []GaosTest

	file, err := ioutil.ReadFile("gaos_tests.json")

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(file, &gaosTests)

	if err != nil {
		return nil, errors.Wrap(err, "Unable to parse scenario file")
	}

	return gaosTests, nil
}

func (gaosTest *GaosTest) circuitBreakerTest() {
	gaos := &runner.Runner{
		Service : map[string]*runner.Service{
			"search" : {
				Port: gaosTest.DeepBackendPort,
				Path: map[string]runner.Path{
					gaosTest.DeepBackendUrl :
						{
							Scenario: "error",
							Method: gaosTest.DeepBackendHttpMethod,
						},
				}},
		},
		Scenario: map[string]*runner.Scenario{
			"error" : {
				Name: "always returns 500",
				Accept: runner.Action{
					Status: 500,
					Result: runner.Result{
						Type: "static",
						Content: map[string]string{
							"description" : "This response should return 500",
						},
					},
				},
			},
		},
		Metrics : runner.Metrics{EndpointCallCounts: map[string]int{}},
	}

	go gaos.Run()

	time.Sleep(2 * time.Second)

	var url = fmt.Sprintf("http://%s:%d%s", gaosTest.MockContainerName, gaosTest.ApiPort, gaosTest.ApiUrl)

	callEndPointMultipleTimesAndValidateCircuitCount(gaosTest, gaos, url, 10, false)
	
	callEndPointMultipleTimesAndValidateCircuitCount(gaosTest, gaos, url, 10, true)

	gaos.ShutdownServers()
}

func callEndPointMultipleTimesAndValidateCircuitCount(gaosTest *GaosTest, gaos *runner.Runner, url string, callCount int, circuitOpen bool) {

	currentGaosEndPointCalledCountStart := gaos.Metrics.GetEndpointCallCount(gaosTest.DeepBackendHttpMethod, gaosTest.DeepBackendCountCheckPath)

	var wg sync.WaitGroup
	wg.Add(callCount)

	for i := 0; i < callCount; i++ {
		go func() {
			callEndPointAndCountCircuits(gaosTest.T, url)
			wg.Done()
		}()
	}

	wg.Wait()

	currentGaosEndPointCalledCountEnd := gaos.Metrics.GetEndpointCallCount(gaosTest.DeepBackendHttpMethod, gaosTest.DeepBackendCountCheckPath)

	if circuitOpen {
		assert.Equal(gaosTest.T, currentGaosEndPointCalledCountEnd-currentGaosEndPointCalledCountStart, 0)
	} else {
		assert.Equal(gaosTest.T, currentGaosEndPointCalledCountEnd-currentGaosEndPointCalledCountStart, callCount * 2)
	}
}

func callEndPointAndCountCircuits(t *testing.T, url string) {
	res, err := http.Get(url)

	assert.Nil(t, err)
	assert.NotNil(t, res)

	if res != nil {
		assert.Equal(t, res.StatusCode, 500)

		var got ApplicationError
		err = json.NewDecoder(res.Body).Decode(&got)
	}

}
