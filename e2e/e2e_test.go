package e2e_test

import (
	"math"
	"testing"
)

const (
	expectedTotal   = 271 // 79 hail + 149 tornado + 43 wind (NOAA SPC 2024-04-26)
	expectedHail    = 79
	expectedTornado = 149
	expectedWind    = 43

	msgAggregationsNil = "aggregations is nil"
)

// fixtureTimeRange matches the mock server's fixture date (240426 → 2024-04-26).
// Using the exact fixture date instead of a wide 2020–2030 window makes tests
// resilient to stale data from other dates that may exist in the database.
const fixtureTimeRange = `timeRange: { from: "2024-04-26T00:00:00Z", to: "2024-04-27T00:00:00Z" }`

func TestServicesHealthy(t *testing.T) {
	waitForHealthy(t, "api", apiURL())
	waitForHealthy(t, "collector", collectorURL())
	waitForHealthy(t, "etl", etlURL())
}

func TestDataPropagation(t *testing.T) {
	ensureDataPropagated(t)
}

func TestReportCounts(t *testing.T) {
	ensureDataPropagated(t)

	query := `{
		stormReports(filter: { ` + fixtureTimeRange + ` }) {
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
		t.Fatal(msgAggregationsNil)
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
		stormReports(filter: { ` + fixtureTimeRange + ` }) {
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
		t.Fatal(msgAggregationsNil)
	}
	states := sr.Aggregations.ByState

	stateMap := map[string]int{}
	for _, s := range states {
		stateMap[s.State] = s.Count
		if len(s.Counties) == 0 {
			t.Errorf("state %s has no county breakdown", s.State)
		}
	}

	if len(stateMap) != 11 {
		t.Errorf("expected 11 states, got %d: %v", len(stateMap), stateMap)
	}
	if stateMap["NE"] != 100 {
		t.Errorf("NE count = %d, want 100", stateMap["NE"])
	}
	if stateMap["IA"] != 69 {
		t.Errorf("IA count = %d, want 69", stateMap["IA"])
	}
	if stateMap["TX"] != 39 {
		t.Errorf("TX count = %d, want 39", stateMap["TX"])
	}
}

func TestReportEnrichment(t *testing.T) {
	ensureDataPropagated(t)

	query := `{
		stormReports(filter: { ` + fixtureTimeRange + ` }) {
			reports {
				id eventType
				measurement { magnitude unit severity }
				sourceOffice timeBucket processedAt
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
			` + fixtureTimeRange + `
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

	if sr.TotalCount < 1 {
		t.Fatalf("expected at least 1 San Saba hail report, got %d", sr.TotalCount)
	}

	// Find the specific 1.25" report among San Saba results.
	var r stormReport
	found := false
	for _, rpt := range sr.Reports {
		if rpt.Measurement.Magnitude == 1.25 {
			r = rpt
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected a San Saba hail report with magnitude 1.25")
	}
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
		stormReports(filter: { ` + fixtureTimeRange + ` }) {
			totalCount
			aggregations {
				byHour { bucket count }
			}
		}
	}`

	result := graphQLQuery(t, query)
	sr := result.Data.StormReports

	if sr.Aggregations == nil {
		t.Fatal(msgAggregationsNil)
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
			` + fixtureTimeRange + `
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
		stormReports(filter: { ` + fixtureTimeRange + ` }) {
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

func TestPagination(t *testing.T) {
	ensureDataPropagated(t)

	// First page: limit 5, offset 0
	page1Query := `{
		stormReports(filter: {
			` + fixtureTimeRange + `
			limit: 5
			offset: 0
		}) {
			totalCount hasMore
			reports { id }
		}
	}`

	r1 := graphQLQuery(t, page1Query)
	sr1 := r1.Data.StormReports

	if sr1.TotalCount != expectedTotal {
		t.Errorf("page 1 totalCount = %d, want %d", sr1.TotalCount, expectedTotal)
	}
	if !sr1.HasMore {
		t.Error("page 1 hasMore should be true")
	}
	if len(sr1.Reports) != 5 {
		t.Errorf("page 1 reports = %d, want 5", len(sr1.Reports))
	}

	// Second page: limit 5, offset 5
	page2Query := `{
		stormReports(filter: {
			` + fixtureTimeRange + `
			limit: 5
			offset: 5
		}) {
			totalCount hasMore
			reports { id }
		}
	}`

	r2 := graphQLQuery(t, page2Query)
	sr2 := r2.Data.StormReports

	if len(sr2.Reports) != 5 {
		t.Errorf("page 2 reports = %d, want 5", len(sr2.Reports))
	}

	// Pages must return different reports.
	ids := map[string]bool{}
	for _, r := range sr1.Reports {
		ids[r.ID] = true
	}
	for _, r := range sr2.Reports {
		if ids[r.ID] {
			t.Errorf("page 2 contains duplicate ID %s from page 1", r.ID)
		}
	}
}

func TestSeverityFilter(t *testing.T) {
	ensureDataPropagated(t)

	query := `{
		stormReports(filter: {
			` + fixtureTimeRange + `
			severity: [SEVERE]
		}) {
			totalCount
			reports { measurement { severity } }
		}
	}`

	result := graphQLQuery(t, query)
	sr := result.Data.StormReports

	if sr.TotalCount == 0 {
		t.Fatal("expected at least one severe report")
	}
	if sr.TotalCount >= expectedTotal {
		t.Errorf("severity filter should narrow results: got %d/%d", sr.TotalCount, expectedTotal)
	}
	for _, r := range sr.Reports {
		if r.Measurement.Severity == nil || *r.Measurement.Severity != "severe" {
			sev := "<nil>"
			if r.Measurement.Severity != nil {
				sev = *r.Measurement.Severity
			}
			t.Errorf("filtered report has severity %q, want severe", sev)
		}
	}
}

func TestSortByMagnitude(t *testing.T) {
	ensureDataPropagated(t)

	query := `{
		stormReports(filter: {
			` + fixtureTimeRange + `
			eventTypes: [HAIL]
			sortBy: MAGNITUDE
			sortOrder: DESC
			limit: 10
		}) {
			reports { measurement { magnitude } }
		}
	}`

	result := graphQLQuery(t, query)
	reports := result.Data.StormReports.Reports

	if len(reports) < 2 {
		t.Fatal("expected at least 2 hail reports")
	}

	for i := 1; i < len(reports); i++ {
		prev := reports[i-1].Measurement.Magnitude
		curr := reports[i].Measurement.Magnitude
		if prev < curr {
			t.Errorf("reports not sorted DESC by magnitude: index %d (%.2f) < index %d (%.2f)", i-1, prev, i, curr)
		}
	}
}

func TestGeoRadiusFilter(t *testing.T) {
	ensureDataPropagated(t)

	// Use approximate center of Nebraska (NE has 100 reports in mock data).
	// A 50-mile radius should return some but not all NE reports.
	query := `{
		stormReports(filter: {
			` + fixtureTimeRange + `
			near: { lat: 41.0, lon: -99.0, radiusMiles: 50 }
		}) {
			totalCount
			reports { geo { lat lon } }
		}
	}`

	result := graphQLQuery(t, query)
	sr := result.Data.StormReports

	if sr.TotalCount == 0 {
		t.Fatal("expected at least one report within 50 miles of central NE")
	}
	if sr.TotalCount >= expectedTotal {
		t.Errorf("geo filter should narrow results: got %d/%d", sr.TotalCount, expectedTotal)
	}

	// Verify all returned reports are within ~50 miles of the center.
	const maxMiles = 55.0 // small tolerance for floating-point
	for _, r := range sr.Reports {
		dist := haversine(41.0, -99.0, r.Geo.Lat, r.Geo.Lon)
		if dist > maxMiles {
			t.Errorf("report at (%.4f, %.4f) is %.1f miles away, exceeds %.1f",
				r.Geo.Lat, r.Geo.Lon, dist, maxMiles)
		}
	}
}

// haversine returns the great-circle distance in miles between two points.
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusMiles = 3959.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	lat1r := lat1 * math.Pi / 180
	lat2r := lat2 * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1r)*math.Cos(lat2r)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadiusMiles * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
