package e2e_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// dataReady gates all data-dependent tests. It polls the GraphQL API once
// (via sync.Once) and fails the calling test if data never appears.
var dataReady sync.Once
var dataReadyErr error

func ensureDataPropagated(t *testing.T) {
	t.Helper()
	dataReady.Do(func() {
		waitForHealthy(t, "api", apiURL(), 60*time.Second)

		query := `{ stormReports(filter: { timeRange: { from: "2020-01-01T00:00:00Z", to: "2030-01-01T00:00:00Z" } }) { totalCount } }`
		deadline := time.Now().Add(120 * time.Second)
		var lastCount int
		for time.Now().Before(deadline) {
			result := graphQLQuery(t, query)
			lastCount = result.Data.StormReports.TotalCount
			if lastCount >= 9 {
				t.Logf("data propagated: %d records found", lastCount)
				return
			}
			t.Logf("waiting for data propagation: %d/9 records", lastCount)
			time.Sleep(5 * time.Second)
		}
		dataReadyErr = fmt.Errorf("data did not propagate: got %d/9 records", lastCount)
	})
	if dataReadyErr != nil {
		t.Fatal(dataReadyErr)
	}
}

// Service URLs with env var overrides for flexibility.
func apiURL() string {
	if v := os.Getenv("API_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

func collectorURL() string {
	if v := os.Getenv("COLLECTOR_URL"); v != "" {
		return v
	}
	return "http://localhost:3000"
}

func etlURL() string {
	if v := os.Getenv("ETL_URL"); v != "" {
		return v
	}
	return "http://localhost:8081"
}

// waitForHealthy polls a /healthz endpoint until it returns 200 or the timeout expires.
func waitForHealthy(t *testing.T, name, url string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	endpoint := url + "/healthz"

	for time.Now().Before(deadline) {
		resp, err := http.Get(endpoint)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				t.Logf("%s is healthy", name)
				return
			}
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("%s did not become healthy within %s (endpoint: %s)", name, timeout, endpoint)
}

// waitForReady polls a /readyz endpoint until it returns 200 or the timeout expires.
func waitForReady(t *testing.T, name, url string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	endpoint := url + "/readyz"

	for time.Now().Before(deadline) {
		resp, err := http.Get(endpoint)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				t.Logf("%s is ready", name)
				return
			}
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("%s did not become ready within %s (endpoint: %s)", name, timeout, endpoint)
}

// graphQLQuery executes a GraphQL query against the API and returns the parsed response.
func graphQLQuery(t *testing.T, query string) graphQLResponse {
	t.Helper()

	payload := fmt.Sprintf(`{"query":%q}`, query)
	resp, err := http.Post(
		apiURL()+"/query",
		"application/json",
		strings.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("GraphQL request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading GraphQL response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GraphQL returned status %d: %s", resp.StatusCode, string(body))
	}

	var result graphQLResponse
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("unmarshaling GraphQL response: %v\nbody: %s", err, string(body))
	}

	if len(result.Errors) > 0 {
		t.Fatalf("GraphQL errors: %v", result.Errors)
	}

	return result
}

// assertReportEnriched checks that all ETL enrichment fields are present on a report.
func assertReportEnriched(t *testing.T, r stormReport) {
	t.Helper()
	if r.ID == "" {
		t.Error("report has empty ID")
	}
	if r.Measurement.Unit == "" {
		t.Errorf("report %s has empty measurement.unit", r.ID)
	}
	if r.TimeBucket == "" {
		t.Errorf("report %s has empty timeBucket", r.ID)
	}
	if r.ProcessedAt == "" {
		t.Errorf("report %s has empty processedAt", r.ID)
	}
	if r.Location.State == "" {
		t.Errorf("report %s has empty state", r.ID)
	}
	if r.Location.County == "" {
		t.Errorf("report %s has empty county", r.ID)
	}
	if r.Geo.Lat == 0 && r.Geo.Lon == 0 {
		t.Errorf("report %s has zero geo coordinates", r.ID)
	}
}

// --- Response types ---

type graphQLResponse struct {
	Data   stormReportsData `json:"data"`
	Errors []graphQLError   `json:"errors"`
}

type graphQLError struct {
	Message string `json:"message"`
}

type stormReportsData struct {
	StormReports stormReportsResult `json:"stormReports"`
}

type stormReportsResult struct {
	TotalCount   int               `json:"totalCount"`
	HasMore      bool              `json:"hasMore"`
	Reports      []stormReport     `json:"reports"`
	Aggregations *stormAggregations `json:"aggregations"`
	Meta         *queryMeta        `json:"meta"`
}

type stormAggregations struct {
	TotalCount  int              `json:"totalCount"`
	ByEventType []eventTypeGroup `json:"byEventType"`
	ByState     []stateGroup     `json:"byState"`
	ByHour      []timeGroup      `json:"byHour"`
}

type queryMeta struct {
	LastUpdated    *string `json:"lastUpdated"`
	DataLagMinutes *int    `json:"dataLagMinutes"`
}

type stormReport struct {
	ID               string      `json:"id"`
	EventType        string      `json:"eventType"`
	Geo              geo         `json:"geo"`
	Measurement      measurement `json:"measurement"`
	BeginTime        string      `json:"beginTime"`
	EndTime          string      `json:"endTime"`
	Source           string      `json:"source"`
	SourceOffice     string      `json:"sourceOffice"`
	Location         location    `json:"location"`
	Comments         string      `json:"comments"`
	TimeBucket       string      `json:"timeBucket"`
	ProcessedAt      string      `json:"processedAt"`
	FormattedAddress string      `json:"formattedAddress"`
	PlaceName        string      `json:"placeName"`
	GeoConfidence    float64     `json:"geoConfidence"`
	GeoSource        string      `json:"geoSource"`
}

type measurement struct {
	Magnitude float64 `json:"magnitude"`
	Unit      string  `json:"unit"`
	Severity  *string `json:"severity"`
}

type geo struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type location struct {
	Raw       string   `json:"raw"`
	Name      string   `json:"name"`
	Distance  *float64 `json:"distance"`
	Direction *string  `json:"direction"`
	State     string   `json:"state"`
	County    string   `json:"county"`
}

type eventTypeGroup struct {
	EventType      string       `json:"eventType"`
	Count          int          `json:"count"`
	MaxMeasurement *measurement `json:"maxMeasurement"`
}

type stateGroup struct {
	State    string        `json:"state"`
	Count    int           `json:"count"`
	Counties []countyGroup `json:"counties"`
}

type countyGroup struct {
	County string `json:"county"`
	Count  int    `json:"count"`
}

type timeGroup struct {
	Bucket string `json:"bucket"`
	Count  int    `json:"count"`
}
