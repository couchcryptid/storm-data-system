package e2e_test

import (
	"testing"
	"time"
)

const (
	expectedTotal   = 9 // 3 hail + 3 tornado + 3 wind
	expectedHail    = 3
	expectedTornado = 3
	expectedWind    = 3
	healthTimeout   = 60 * time.Second
)

const wideTimeRange = `timeRange: { from: "2020-01-01T00:00:00Z", to: "2030-01-01T00:00:00Z" }`

func TestServicesHealthy(t *testing.T) {
	waitForHealthy(t, "api", apiURL(), healthTimeout)
	waitForHealthy(t, "collector", collectorURL(), healthTimeout)
	waitForHealthy(t, "etl", etlURL(), healthTimeout)
}

func TestDataPropagation(t *testing.T) {
	ensureDataPropagated(t)
}

func TestReportCounts(t *testing.T) {
	ensureDataPropagated(t)

	query := `{
		stormReports(filter: { ` + wideTimeRange + ` }) {
			totalCount
			aggregations {
				byEventType { eventType count maxMeasurement { magnitude unit } }
			}
		}
	}`

	result := graphQLQuery(t, query)
	sr := result.Data.StormReports

	if sr.TotalCount != expectedTotal {
		t.Errorf("totalCount = %d, want %d", sr.TotalCount, expectedTotal)
	}

	if sr.Aggregations == nil {
		t.Fatal("aggregations is nil")
	}

	typeCounts := map[string]int{}
	for _, g := range sr.Aggregations.ByEventType {
		typeCounts[g.EventType] = g.Count
	}
	if typeCounts["hail"] != expectedHail {
		t.Errorf("hail count = %d, want %d", typeCounts["hail"], expectedHail)
	}
	if typeCounts["tornado"] != expectedTornado {
		t.Errorf("tornado count = %d, want %d", typeCounts["tornado"], expectedTornado)
	}
	if typeCounts["wind"] != expectedWind {
		t.Errorf("wind count = %d, want %d", typeCounts["wind"], expectedWind)
	}
}

func TestStateAggregations(t *testing.T) {
	ensureDataPropagated(t)

	query := `{
		stormReports(filter: { ` + wideTimeRange + ` }) {
			aggregations {
				byState {
					state count
					counties { county count }
				}
			}
		}
	}`

	result := graphQLQuery(t, query)
	sr := result.Data.StormReports

	if sr.Aggregations == nil {
		t.Fatal("aggregations is nil")
	}
	states := sr.Aggregations.ByState

	stateMap := map[string]int{}
	for _, s := range states {
		stateMap[s.State] = s.Count
		if len(s.Counties) == 0 {
			t.Errorf("state %s has no county breakdown", s.State)
		}
	}

	if len(stateMap) != 3 {
		t.Errorf("expected 3 states, got %d: %v", len(stateMap), stateMap)
	}
	if stateMap["TX"] != 5 {
		t.Errorf("TX count = %d, want 5", stateMap["TX"])
	}
	if stateMap["OK"] != 3 {
		t.Errorf("OK count = %d, want 3", stateMap["OK"])
	}
	if stateMap["NE"] != 1 {
		t.Errorf("NE count = %d, want 1", stateMap["NE"])
	}
}

func TestReportEnrichment(t *testing.T) {
	ensureDataPropagated(t)

	query := `{
		stormReports(filter: { ` + wideTimeRange + ` }) {
			reports {
				id eventType
				measurement { magnitude unit severity }
				sourceOffice timeBucket processedAt
				formattedAddress placeName geoConfidence geoSource
				geo { lat lon }
				location { raw name state county }
			}
		}
	}`

	result := graphQLQuery(t, query)
	reports := result.Data.StormReports.Reports

	if len(reports) == 0 {
		t.Fatal("no reports returned")
	}

	for _, r := range reports {
		assertReportEnriched(t, r)
	}
}

func TestSpotCheckHailReport(t *testing.T) {
	ensureDataPropagated(t)

	query := `{
		stormReports(filter: {
			` + wideTimeRange + `
			eventTypes: [HAIL]
			counties: ["San Saba"]
		}) {
			totalCount
			reports {
				eventType
				measurement { magnitude unit }
				sourceOffice
				location { raw name state county direction distance }
			}
		}
	}`

	result := graphQLQuery(t, query)
	sr := result.Data.StormReports

	if sr.TotalCount != 1 {
		t.Fatalf("expected 1 San Saba hail report, got %d", sr.TotalCount)
	}

	r := sr.Reports[0]
	if r.EventType != "hail" {
		t.Errorf("eventType = %q, want hail", r.EventType)
	}
	if r.Measurement.Magnitude != 1.25 {
		t.Errorf("measurement.magnitude = %f, want 1.25", r.Measurement.Magnitude)
	}
	if r.Measurement.Unit != "in" {
		t.Errorf("measurement.unit = %q, want in", r.Measurement.Unit)
	}
	if r.SourceOffice != "SJT" {
		t.Errorf("sourceOffice = %q, want SJT", r.SourceOffice)
	}
	if r.Location.Name != "Chappel" {
		t.Errorf("location.name = %q, want Chappel", r.Location.Name)
	}
	if r.Location.State != "TX" {
		t.Errorf("location.state = %q, want TX", r.Location.State)
	}
	if r.Location.Direction != nil && *r.Location.Direction != "ESE" {
		t.Errorf("location.direction = %q, want ESE", *r.Location.Direction)
	}
}

func TestHourlyAggregation(t *testing.T) {
	ensureDataPropagated(t)

	query := `{
		stormReports(filter: { ` + wideTimeRange + ` }) {
			totalCount
			aggregations {
				byHour { bucket count }
			}
		}
	}`

	result := graphQLQuery(t, query)
	sr := result.Data.StormReports

	if sr.Aggregations == nil {
		t.Fatal("aggregations is nil")
	}

	if len(sr.Aggregations.ByHour) == 0 {
		t.Fatal("expected at least one hourly bucket")
	}

	hourTotal := 0
	for _, h := range sr.Aggregations.ByHour {
		if h.Bucket == "" {
			t.Error("hourly bucket has empty timestamp")
		}
		hourTotal += h.Count
	}

	if hourTotal != sr.TotalCount {
		t.Errorf("hourly bucket total = %d, totalCount = %d", hourTotal, sr.TotalCount)
	}
}

func TestEventTypeFilter(t *testing.T) {
	ensureDataPropagated(t)

	query := `{
		stormReports(filter: {
			` + wideTimeRange + `
			eventTypes: [TORNADO]
		}) {
			totalCount
			reports { eventType }
		}
	}`

	result := graphQLQuery(t, query)
	sr := result.Data.StormReports

	if sr.TotalCount != expectedTornado {
		t.Errorf("tornado filter totalCount = %d, want %d", sr.TotalCount, expectedTornado)
	}
	for _, r := range sr.Reports {
		if r.EventType != "tornado" {
			t.Errorf("filtered report has eventType %q, want tornado", r.EventType)
		}
	}
}

func TestMeta(t *testing.T) {
	ensureDataPropagated(t)

	query := `{
		stormReports(filter: { ` + wideTimeRange + ` }) {
			meta {
				lastUpdated
				dataLagMinutes
			}
		}
	}`

	result := graphQLQuery(t, query)
	sr := result.Data.StormReports

	if sr.Meta == nil {
		t.Fatal("meta is nil")
	}
	if sr.Meta.LastUpdated == nil {
		t.Error("meta.lastUpdated is nil")
	}
	if sr.Meta.DataLagMinutes == nil {
		t.Error("meta.dataLagMinutes is nil")
	}
}
