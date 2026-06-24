package devices

import (
	"context"
	"crypto/rand"
	"encoding/csv"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/internal/download"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/timepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

// newDownloadRouter builds a download.Router with a fresh random key for use
// in tests. The base URL is a path-only "/download" so generated URLs are
// root-relative, which httptest accepts directly.
func newDownloadRouter(t *testing.T) *download.Router {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	return download.NewRouter(
		download.NewHMACSigner(key),
		download.WithBaseURL("/download"),
		download.WithTTL(time.Hour),
	)
}

// newDownloadedServer builds a Server with a fresh download.Router fully wired
// up: the Server generates URLs through it, and the Server itself is registered
// against it under DownloadType. Returns both so tests can call Router.ServeHTTP
// directly.
func newDownloadedServer(t *testing.T, m Model, opts ...Option) (*Server, *download.Router) {
	t.Helper()
	rt := newDownloadRouter(t)
	opts = append(opts, WithURLGenerator(rt))
	s := NewServer(m, opts...)
	rt.Handle(DownloadType, s)
	return s, rt
}

func TestServer_DownloadDevicesHTTPHandler(t *testing.T) {
	now := time.Unix(0, 0)
	n := node.New("test")

	meterDevice := meterpb.NewModel()
	_, _ = meterDevice.UpdateMeterReading(&meterpb.MeterReading{Usage: 200})
	n.Announce("d1",
		node.HasServer(meterpb.RegisterMeterApiServer, meterpb.MeterApiServer(meterpb.NewModelServer(meterDevice))),
		node.HasServer(meterpb.RegisterMeterInfoServer, meterpb.MeterInfoServer(&meterpb.InfoServer{MeterReading: &meterpb.MeterReadingSupport{
			UsageUnit: "tests per second",
		}})),
		node.HasTrait(meterpb.TraitName),
		node.HasMetadata(&metadatapb.Metadata{Location: &metadatapb.Metadata_Location{Floor: "01"}}),
	)

	airTempDevice := airtemperaturepb.NewModel()
	_, _ = airTempDevice.UpdateAirTemperature(&airtemperaturepb.AirTemperature{
		TemperatureGoal:    &airtemperaturepb.AirTemperature_TemperatureSetPoint{TemperatureSetPoint: &typespb.Temperature{ValueCelsius: 23.5}},
		AmbientTemperature: &typespb.Temperature{ValueCelsius: 19.2},
		AmbientHumidity:    proto.Float32(62.1),
	})
	n.Announce("d2",
		node.HasServer(airtemperaturepb.RegisterAirTemperatureApiServer, airtemperaturepb.AirTemperatureApiServer(airtemperaturepb.NewModelServer(airTempDevice))),
		node.HasTrait(trait.AirTemperature),
		node.HasMetadata(&metadatapb.Metadata{Location: &metadatapb.Metadata_Location{Floor: "02"}}),
	)

	s, rt := newDownloadedServer(t, n, WithNow(func() time.Time { return now }))

	devicesUrl, err := s.GetDownloadDevicesUrl(context.Background(), &devicespb.GetDownloadDevicesUrlRequest{
		Query: &devicespb.Device_Query{Conditions: []*devicespb.Device_Query_Condition{
			{Field: "metadata.location.floor", Value: &devicespb.Device_Query_Condition_StringEqual{StringEqual: "01"}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", devicesUrl.Url, nil)
	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != 200 {
		t.Fatalf("HTTP status code: expected 200, got %d", res.StatusCode)
	}
	ct := newCsvTester(t, res.Body)
	assertHeaderOrder(t, ct.headerRow)
	// the query should include this
	ct.assertCellValue("d1", "name", "d1")
	ct.assertCellValue("d1", "md.location.floor", "01")
	ct.assertCellValue("d1", "meter.usage", "200.000")
	ct.assertCellValue("d1", "meter.unit", "tests per second")
	// the query should not include this
	ct.assertNoRow("d2")
}

func TestServer_DownloadDevicesHTTPHandler_validation(t *testing.T) {
	t.Run("expired", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			rt := newDownloadRouter(t)
			s := NewServer(node.New("test"), WithURLGenerator(rt))

			devicesUrl, err := s.GetDownloadDevicesUrl(context.Background(), &devicespb.GetDownloadDevicesUrlRequest{})
			if err != nil {
				t.Fatal(err)
			}

			// Cross the TTL boundary.
			time.Sleep(time.Hour + time.Minute)

			req := httptest.NewRequest("GET", devicesUrl.Url, nil)
			rec := httptest.NewRecorder()
			rt.ServeHTTP(rec, req)

			res := rec.Result()
			if res.StatusCode != http.StatusUnauthorized {
				t.Fatalf("HTTP status code: expected %d, got %d", http.StatusUnauthorized, res.StatusCode)
			}
		})
	})

	t.Run("no token", func(t *testing.T) {
		rt := newDownloadRouter(t)
		// Path matches the mount prefix but has no token segment after it.
		req := httptest.NewRequest("GET", "/download/", nil)
		rec := httptest.NewRecorder()
		rt.ServeHTTP(rec, req)

		res := rec.Result()
		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("HTTP status code: expected %d, got %d", http.StatusNotFound, res.StatusCode)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		rt := newDownloadRouter(t)
		req := httptest.NewRequest("GET", "/download/not-a-valid-token", nil)
		rec := httptest.NewRecorder()
		rt.ServeHTTP(rec, req)

		res := rec.Result()
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("HTTP status code: expected %d, got %d", http.StatusUnauthorized, res.StatusCode)
		}
	})

	t.Run("change of key", func(t *testing.T) {
		// Two independent Routers means two independent random HMAC keys.
		signRouter := newDownloadRouter(t)
		verifyRouter := newDownloadRouter(t)
		signer := NewServer(node.New("test"), WithURLGenerator(signRouter))

		devicesUrl, err := signer.GetDownloadDevicesUrl(context.Background(), &devicespb.GetDownloadDevicesUrlRequest{})
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("GET", devicesUrl.Url, nil)
		rec := httptest.NewRecorder()
		verifyRouter.ServeHTTP(rec, req)

		res := rec.Result()
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("HTTP status code: expected %d, got %d", http.StatusUnauthorized, res.StatusCode)
		}
	})

	t.Run("downloads disabled when no router", func(t *testing.T) {
		s := NewServer(node.New("test"))
		_, err := s.GetDownloadDevicesUrl(context.Background(), &devicespb.GetDownloadDevicesUrlRequest{})
		if err == nil {
			t.Fatal("expected error from GetDownloadDevicesUrl, got nil")
		}
		if got := status.Code(err); got != codes.Unavailable {
			t.Fatalf("expected codes.Unavailable, got %s: %v", got, err)
		}
	})
}

type csvTester struct {
	t           *testing.T
	r           *csv.Reader
	headerRow   []string
	headerIndex map[string]int

	rows       [][]string
	rowsByName map[string][]string
}

func newCsvTester(t *testing.T, r io.Reader) *csvTester {
	t.Helper()
	csvReader := csv.NewReader(r)
	header, err := csvReader.Read()
	if err != nil {
		t.Fatalf("CSV header read error: %v", err)
	}
	headerIndex := make(map[string]int, len(header))
	for i, col := range header {
		headerIndex[col] = i
	}

	if _, ok := headerIndex["name"]; !ok {
		t.Fatalf("expected name column in header")
	}

	ct := &csvTester{
		t:           t,
		r:           csvReader,
		headerRow:   header,
		headerIndex: headerIndex,
		rowsByName:  make(map[string][]string),
	}

	var i int
	for {
		i++
		row, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("CSV row read error at line %d: %v", i, err)
		}
		if len(row) != len(header) {
			t.Errorf("expected %d columns, got %d, for line %d", len(header), len(row), i)
		}
		ct.rows = append(ct.rows, row)
		name := row[ct.headerIndex["name"]]
		ct.rowsByName[name] = row
	}

	return ct
}

func (ct *csvTester) assertCellValue(name, col, want string) {
	ct.t.Helper()
	row, ok := ct.rowsByName[name]
	if !ok {
		ct.t.Errorf("expected row with name %q", name)
	}
	i, ok := ct.headerIndex[col]
	if !ok {
		ct.t.Errorf("expected column %q", col)
	}
	if row[i] != want {
		ct.t.Errorf("expected %q in %q column for row %q, got %q", want, col, name, row[i])
	}
}

func (ct *csvTester) assertNoRow(name string) {
	ct.t.Helper()
	if r, ok := ct.rowsByName[name]; ok {
		ct.t.Errorf("expected no row with name %q, got %v", name, r)
	}
}

// TestServer_DownloadDevicesHTTPHandler_history exercises the historical-data
// path: a meter device with a stub history server and a request carrying a
// History period should render a CSV with a timestamp column and one row per
// historical record.
func TestServer_DownloadDevicesHTTPHandler_history(t *testing.T) {
	n := node.New("test")

	meterDevice := meterpb.NewModel()
	_, _ = meterDevice.UpdateMeterReading(&meterpb.MeterReading{Usage: 0})
	historyStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	records := []*meterpb.MeterReadingRecord{
		{RecordTime: timestamppb.New(historyStart), MeterReading: &meterpb.MeterReading{Usage: 100}},
		{RecordTime: timestamppb.New(historyStart.Add(time.Hour)), MeterReading: &meterpb.MeterReading{Usage: 150}},
		{RecordTime: timestamppb.New(historyStart.Add(2 * time.Hour)), MeterReading: &meterpb.MeterReading{Usage: 220}},
	}
	n.Announce("d1",
		node.HasServer(meterpb.RegisterMeterApiServer, meterpb.MeterApiServer(meterpb.NewModelServer(meterDevice))),
		node.HasServer(meterpb.RegisterMeterInfoServer, meterpb.MeterInfoServer(&meterpb.InfoServer{MeterReading: &meterpb.MeterReadingSupport{
			UsageUnit: "tests per second",
		}})),
		node.HasServer(meterpb.RegisterMeterHistoryServer, meterpb.MeterHistoryServer(&stubMeterHistory{records: records})),
		node.HasTrait(meterpb.TraitName),
		node.HasMetadata(&metadatapb.Metadata{Location: &metadatapb.Metadata_Location{Floor: "01"}}),
	)

	dr := newDownloadRouter(t)
	s := NewServer(n, WithURLGenerator(dr))
	dr.Handle(DownloadType, s)

	devicesUrl, err := s.GetDownloadDevicesUrl(context.Background(), &devicespb.GetDownloadDevicesUrlRequest{
		History: &timepb.Period{
			StartTime: timestamppb.New(historyStart),
			EndTime:   timestamppb.New(historyStart.Add(3 * time.Hour)),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", devicesUrl.Url, nil)
	rec := httptest.NewRecorder()
	dr.ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != 200 {
		t.Fatalf("HTTP status code: expected 200, got %d", res.StatusCode)
	}
	ct := newCsvTester(t, res.Body)

	// History mode must include a timestamp column, and it should come first.
	tsIdx, ok := ct.headerIndex["timestamp"]
	if !ok {
		t.Fatalf("expected timestamp column in history mode, headers=%v", ct.headerRow)
	}
	if tsIdx != 0 {
		t.Fatalf("expected timestamp to be first column, got index %d (headers=%v)", tsIdx, ct.headerRow)
	}

	// One row per historical record.
	if len(ct.rows) != len(records) {
		t.Fatalf("expected %d rows, got %d", len(records), len(ct.rows))
	}

	usageIdx, ok := ct.headerIndex["meter.usage"]
	if !ok {
		t.Fatalf("expected meter.usage column, headers=%v", ct.headerRow)
	}
	wantUsage := []string{"100.000", "150.000", "220.000"}
	for i, row := range ct.rows {
		if row[usageIdx] != wantUsage[i] {
			t.Errorf("row %d meter.usage: want %q, got %q", i, wantUsage[i], row[usageIdx])
		}
		// Timestamp should parse and lie within the requested period.
		if _, err := time.Parse(time.DateTime, row[tsIdx]); err != nil {
			t.Errorf("row %d timestamp %q does not parse as %q: %v", i, row[tsIdx], time.DateTime, err)
		}
	}
}

type stubMeterHistory struct {
	meterpb.UnimplementedMeterHistoryServer
	records []*meterpb.MeterReadingRecord
}

func (s *stubMeterHistory) ListMeterReadingHistory(_ context.Context, _ *meterpb.ListMeterReadingHistoryRequest) (*meterpb.ListMeterReadingHistoryResponse, error) {
	return &meterpb.ListMeterReadingHistoryResponse{MeterReadingRecords: s.records}, nil
}

// TestServer_DownloadDevicesHTTPHandler_mux mounts the shared download.Router
// on an http.ServeMux at "/download/", brings it up via httptest, and fetches
// the URL returned by GetDownloadDevicesUrl over real HTTP. Exercises the
// mount-path wiring end-to-end.
func TestServer_DownloadDevicesHTTPHandler_mux(t *testing.T) {
	n := node.New("test")
	meterDevice := meterpb.NewModel()
	_, _ = meterDevice.UpdateMeterReading(&meterpb.MeterReading{Usage: 42})
	n.Announce("d1",
		node.HasServer(meterpb.RegisterMeterApiServer, meterpb.MeterApiServer(meterpb.NewModelServer(meterDevice))),
		node.HasTrait(meterpb.TraitName),
		node.HasMetadata(&metadatapb.Metadata{}),
	)

	dr := newDownloadRouter(t)
	s := NewServer(n, WithURLGenerator(dr))
	dr.Handle(DownloadType, s)

	mux := http.NewServeMux()
	mux.Handle("/download/", dr)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	devicesUrl, err := s.GetDownloadDevicesUrl(context.Background(), &devicespb.GetDownloadDevicesUrlRequest{})
	if err != nil {
		t.Fatal(err)
	}

	// devicesUrl.Url is path-only because the base URL has no scheme/host.
	resp, err := http.Get(srv.URL + devicesUrl.Url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("HTTP status code: expected 200, got %d", resp.StatusCode)
	}
	ct := newCsvTester(t, resp.Body)
	ct.assertCellValue("d1", "name", "d1")
	ct.assertCellValue("d1", "meter.usage", "42.000")
}

// TestServer_DownloadDevicesHTTPHandler_traitGetError verifies that when a
// trait's get function returns an error (e.g. the trait is advertised but no
// server is registered for it), the failure is surfaced as "ERR: ..." in the
// first column for that trait rather than aborting the response.
func TestServer_DownloadDevicesHTTPHandler_traitGetError(t *testing.T) {
	n := node.New("test")
	// d1 advertises the meter trait but has no MeterApiServer registered.
	// Any get against it returns codes.Unimplemented.
	n.Announce("d1",
		node.HasTrait(meterpb.TraitName),
		node.HasMetadata(&metadatapb.Metadata{Location: &metadatapb.Metadata_Location{Floor: "07"}}),
	)

	dr := newDownloadRouter(t)
	s := NewServer(n, WithURLGenerator(dr))
	dr.Handle(DownloadType, s)

	devicesUrl, err := s.GetDownloadDevicesUrl(context.Background(), &devicespb.GetDownloadDevicesUrlRequest{})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", devicesUrl.Url, nil)
	rec := httptest.NewRecorder()
	dr.ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != 200 {
		t.Fatalf("HTTP status code: expected 200, got %d", res.StatusCode)
	}
	ct := newCsvTester(t, res.Body)
	ct.assertCellValue("d1", "name", "d1")
	ct.assertCellValue("d1", "md.location.floor", "07") // unaffected by trait failure
	row, ok := ct.rowsByName["d1"]
	if !ok {
		t.Fatal("missing d1 row")
	}
	idx, ok := ct.headerIndex["meter.usage"]
	if !ok {
		t.Fatalf("expected meter.usage column, headers=%v", ct.headerRow)
	}
	if !strings.HasPrefix(row[idx], "ERR:") {
		t.Errorf("expected meter.usage to start with ERR:, got %q", row[idx])
	}
}

// GetDownloadDevicesUrl rejects unsupported media types, over-long filenames,
// and queries with too many string_in entries with InvalidArgument before any
// signing or routing work happens.
func TestServer_GetDownloadDevicesUrl_validation(t *testing.T) {
	s := NewServer(node.New("test"))

	tooManyStrings := make([]string, 101)
	for i := range tooManyStrings {
		tooManyStrings[i] = "x"
	}

	cases := map[string]*devicespb.GetDownloadDevicesUrlRequest{
		"unsupported media type": {MediaType: "application/json"},
		"filename too long":      {Filename: strings.Repeat("a", 256)},
		"query string_in too big": {
			Query: &devicespb.Device_Query{
				Conditions: []*devicespb.Device_Query_Condition{
					{Field: "metadata.location.floor", Value: &devicespb.Device_Query_Condition_StringIn{
						StringIn: &devicespb.Device_Query_StringList{Strings: tooManyStrings},
					}},
				},
			},
		},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := s.GetDownloadDevicesUrl(context.Background(), req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if got := status.Code(err); got != codes.InvalidArgument {
				t.Fatalf("expected InvalidArgument, got %s: %v", got, err)
			}
		})
	}
}

func assertHeaderOrder(t *testing.T, row []string) {
	t.Helper()
	if len(row) == 0 {
		t.Fatalf("expected non-empty header row")
	}
	if row[0] != "name" {
		t.Fatalf("expected first column to be 'name', got %q", row[0])
	}

	if len(row) == 1 {
		return // no metadata
	}

	// headers should start with name, then md.* cols, then everything else.
	// There should be no md.name column, as that would duplicate "name" in the first column.
	lastMdIndex := -1
	for i, col := range row {
		// skip the first column which is "name"
		if i == 0 {
			continue
		}
		if lastMdIndex >= 0 {
			if strings.HasPrefix(col, "md.") {
				t.Fatalf("expected md.* cols to be before non-md cols, got %q at %d", col, i)
			}
			continue
		}
		if !strings.HasPrefix(col, "md.") {
			lastMdIndex = i - 1
		}
	}
	if lastMdIndex == -1 {
		// there were no non-md cols
		lastMdIndex = 0
	}
	for i, s := range row[1:lastMdIndex] {
		if s == "md.name" {
			t.Fatalf("expected no md.name column, found at index %d", i+1)
		}
	}

	// headers should be sorted: md.* cols sorted as one group, then non-md cols sorted as the rest
	mdCols := append([]string(nil), row[1:lastMdIndex+1]...)
	nonMdCols := append([]string(nil), row[lastMdIndex+1:]...)
	slices.Sort(mdCols)
	slices.Sort(nonMdCols)
	if diff := cmp.Diff(mdCols, row[1:lastMdIndex+1], cmpopts.EquateEmpty()); diff != "" {
		t.Fatalf("expected md.* cols to be sorted, (-want,+got):\n%v", diff)
	}
	if diff := cmp.Diff(nonMdCols, row[lastMdIndex+1:], cmpopts.EquateEmpty()); diff != "" {
		t.Fatalf("expected non-md cols to be sorted, (-want,+got):\n%v", diff)
	}
}
