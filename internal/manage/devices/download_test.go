package devices

import (
	"context"
	"encoding/csv"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/meter"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/airtemperaturepb"
)

func TestServer_DownloadDevicesHTTPHandler(t *testing.T) {
	now := time.Unix(0, 0)
	n := node.New("test")

	meterDevice := meter.NewModel()
	_, _ = meterDevice.UpdateMeterReading(&meterpb.MeterReading{Usage: 200})
	n.Announce("d1",
		node.HasTrait(
			meter.TraitName,
			node.WithClients(
				meterpb.WrapApi(meter.NewModelServer(meterDevice)),
				meterpb.WrapInfo(&meter.InfoServer{MeterReading: &meterpb.MeterReadingSupport{
					UsageUnit: "tests per second",
				}}),
			),
		),
		node.HasMetadata(&traits.Metadata{Location: &traits.Metadata_Location{Floor: "01"}}),
	)

	airTempDevice := airtemperaturepb.NewModel()
	_, _ = airTempDevice.UpdateAirTemperature(&traits.AirTemperature{
		TemperatureGoal:    &traits.AirTemperature_TemperatureSetPoint{TemperatureSetPoint: &types.Temperature{ValueCelsius: 23.5}},
		AmbientTemperature: &types.Temperature{ValueCelsius: 19.2},
		AmbientHumidity:    proto.Float32(62.1),
	})
	n.Announce("d2",
		node.HasTrait(
			trait.AirTemperature,
			node.WithClients(
				airtemperaturepb.WrapApi(airtemperaturepb.NewModelServer(airTempDevice)),
			),
		),
		node.HasMetadata(&traits.Metadata{Location: &traits.Metadata_Location{Floor: "02"}}),
	)

	s := NewServer(n,
		WithDownloadUrlBase(url.URL{Scheme: "https", Host: "example.com", Path: "/dl/devices"}),
		WithNow(func() time.Time {
			return now
		}),
	)

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
	s.DownloadDevicesHTTPHandler(rec, req)

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
		now := time.Unix(0, 0)
		s := NewServer(
			node.New("test"),
			WithNow(func() time.Time { return now }),
		)

		devicesUrl, err := s.GetDownloadDevicesUrl(context.Background(), &devicespb.GetDownloadDevicesUrlRequest{})
		if err != nil {
			t.Fatal(err)
		}
		now = now.Add(s.downloadExpiry + s.downloadExpiryLeeway + time.Second)

		req := httptest.NewRequest("GET", devicesUrl.Url, nil)
		rec := httptest.NewRecorder()
		s.DownloadDevicesHTTPHandler(rec, req)

		res := rec.Result()
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("HTTP status code: expected %d, got %d", http.StatusUnauthorized, res.StatusCode)
		}
	})

	t.Run("no token", func(t *testing.T) {
		s := NewServer(node.New("test"))
		req := httptest.NewRequest("GET", "/dl/devices", nil)
		rec := httptest.NewRecorder()
		s.DownloadDevicesHTTPHandler(rec, req)

		res := rec.Result()
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("HTTP status code: expected %d, got %d", http.StatusUnauthorized, res.StatusCode)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		s := NewServer(node.New("test"))
		u := s.downloadUrlBase // copy
		if err := writeDownloadToken(&u, "invalid"); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("GET", u.String(), nil)
		rec := httptest.NewRecorder()
		s.DownloadDevicesHTTPHandler(rec, req)

		res := rec.Result()
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("HTTP status code: expected %d, got %d", http.StatusUnauthorized, res.StatusCode)
		}
	})

	t.Run("change of key", func(t *testing.T) {
		s := NewServer(node.New("test"))
		devicesUrl, err := s.GetDownloadDevicesUrl(context.Background(), &devicespb.GetDownloadDevicesUrlRequest{})
		if err != nil {
			t.Fatal(err)
		}
		s.downloadKey = newHMACKeyGen(64) // force a new key
		req := httptest.NewRequest("GET", devicesUrl.Url, nil)
		rec := httptest.NewRecorder()
		s.DownloadDevicesHTTPHandler(rec, req)

		res := rec.Result()
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("HTTP status code: expected %d, got %d", http.StatusUnauthorized, res.StatusCode)
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
