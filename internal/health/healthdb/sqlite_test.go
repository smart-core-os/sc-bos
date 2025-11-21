package healthdb

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/vanti-dev/sc-bos/pkg/gen"
)

func TestDB_records(t *testing.T) {
	db := newDBTester(t)
	want := []Record{
		db.InsertRecord("dev1", "toner", "main1", "aux1"),
		db.InsertRecord("dev1", "toner", "main2", "aux2"),
		db.InsertRecord("dev2", "paper", "main1", "aux1"),
	}
	db.AssertRecords(want...)
}

func TestDB_Read(t *testing.T) {
	db := newDBTester(t)
	all := []Record{
		db.InsertRecord("dev1", "toner", "main1", "aux1"),
		db.InsertRecord("dev2", "paper", "main1", "aux1"),
		db.InsertRecord("dev1", "toner", "main2", "aux2"),
		db.InsertRecord("dev2", "paper", "main2", "aux2"),
		db.InsertRecord("dev1", "paper", "main3", "aux3"),
	}

	db.Run("all", func(t *dbTester) {
		t.AssertRecords(all...)
	})
	db.Run("page asc", func(t *dbTester) {
		page := make([]Record, 2)
		n, err := db.Read(t.Context(), CheckID{}, 0, 0, false, page)
		t.AssertReadResponse(n, err, all[:2], page)
	})
	db.Run("page desc", func(t *dbTester) {
		page := make([]Record, 2)
		n, err := db.Read(t.Context(), CheckID{}, 0, 0, true, page)
		want := []Record{all[4], all[3]}
		t.AssertReadResponse(n, err, want, page)
	})
	db.Run("filter name+id", func(t *dbTester) {
		got := make([]Record, len(all))
		want := []Record{all[0], all[2]}
		n, err := db.Read(t.Context(), CheckID{"dev1", "toner"}, 0, 0, false, got)
		t.AssertReadResponse(n, err, want, got)
	})
	db.Run("filter name", func(t *dbTester) {
		got := make([]Record, len(all))
		want := []Record{all[0], all[2], all[4]}
		n, err := db.Read(t.Context(), CheckID{"dev1", ""}, 0, 0, false, got)
		t.AssertReadResponse(n, err, want, got)
	})
	db.Run("filter from+to", func(t *dbTester) {
		got := make([]Record, len(all))
		want := []Record{all[1], all[2], all[3]}
		from := all[1].ID
		to := all[4].ID
		n, err := db.Read(t.Context(), CheckID{}, from, to, false, got)
		t.AssertReadResponse(n, err, want, got)
	})
	db.Run("filter from+to desc", func(t *dbTester) {
		got := make([]Record, len(all))
		want := []Record{all[3], all[2], all[1]}
		from := all[1].ID
		to := all[4].ID
		n, err := db.Read(t.Context(), CheckID{}, from, to, true, got)
		t.AssertReadResponse(n, err, want, got)
	})
	db.Run("filter from", func(t *dbTester) {
		got := make([]Record, len(all))
		want := []Record{all[2], all[3], all[4]}
		from := all[2].ID
		n, err := db.Read(t.Context(), CheckID{}, from, 0, false, got)
		t.AssertReadResponse(n, err, want, got)
	})
	db.Run("filter to", func(t *dbTester) {
		got := make([]Record, len(all))
		want := []Record{all[0], all[1], all[2]}
		to := all[3].ID
		n, err := db.Read(t.Context(), CheckID{}, 0, to, false, got)
		t.AssertReadResponse(n, err, want, got)
	})
}

func TestDB_Trim(t *testing.T) {
	const (
		nDevices = 2
		nChecks  = 2
		nPer     = 5 // each device+check combo has this many records
	)
	newDB := func(t *testing.T) (*dbTester, []Record) {
		t.Helper()
		db := newDBTester(t)
		all := make([]Record, 0, nDevices*nChecks*nPer)
		// dev1,id1
		// dev2,id1
		// dev1,id2
		// dev2,id2
		// ...
		// dev1,id1
		// dev2,id1
		// etc for nPer
		for range nPer {
			for checkID := range nChecks {
				for devID := range nDevices {
					all = append(all, db.InsertRecord(
						fmt.Sprintf("dev%d", devID+1),
						fmt.Sprintf("id%d", checkID+1),
						"",
						"",
					))
				}
			}
		}
		return db, all
	}
	isDev := func(devID int) func(i int) bool {
		return func(i int) bool {
			return i%nDevices == devID-1
		}
	}
	isCheck := func(checkID int) func(i int) bool {
		return func(i int) bool {
			return (i/nDevices)%nChecks == checkID-1
		}
	}
	isRecordNumRange := func(from, to int) func(i int) bool {
		return func(i int) bool {
			n := i / (nDevices * nChecks)
			return n >= from && n < to
		}
	}
	isBefore := func(t time.Time) func(i int) bool {
		sinceEpoch := t.Sub(dbEpoch)
		beforeIdx := int(sinceEpoch / createCadence)
		return func(i int) bool {
			return i < beforeIdx
		}
	}
	// exclude returns records from all that match none of the given functions.
	// For example, exclude(all, isDev(1), isCheck(2)) returns all records that are not
	// from device 1 and check 2, but will include device 1 check 1, device 2 check 2, etc.
	exclude := func(all []Record, fns ...func(i int) bool) []Record {
		var unmatched []Record
		for i, r := range all {
			matches := true
			for _, fn := range fns {
				if !fn(i) {
					matches = false
					break
				}
			}
			if !matches {
				unmatched = append(unmatched, r)
			}
		}
		return unmatched
	}

	t.Run("empty", func(t *testing.T) {
		db, all := newDB(t)
		n, err := db.Trim(t.Context(), CheckID{}, TrimOptions{})
		if err != nil {
			t.Fatalf("Trim failed: %v", err)
		}
		if n != 0 {
			t.Fatalf("Trim removed %d records, want 0", n)
		}
		db.AssertRecords(all...)
	})
	t.Run("min only", func(t *testing.T) {
		db, all := newDB(t)
		n, err := db.Trim(t.Context(), CheckID{}, TrimOptions{MinCount: 2})
		if err != nil {
			t.Fatalf("Trim failed: %v", err)
		}
		if n != 0 {
			t.Fatalf("Trim removed %d records, want 0", n)
		}
		db.AssertRecords(all...)
	})
	t.Run("max name+id", func(t *testing.T) {
		db, all := newDB(t)
		n, err := db.Trim(t.Context(), CheckID{Name: "dev1", ID: "id1"}, TrimOptions{MaxCount: 3})
		if err != nil {
			t.Fatalf("Trim failed: %v", err)
		}
		if want, got := int64(nPer-3), n; want != got {
			t.Fatalf("Trim removed %d records, want %d", got, want)
		}
		want := exclude(all, isDev(1), isCheck(1), isRecordNumRange(0, 2))
		db.AssertRecords(want...)
	})
	t.Run("max name", func(t *testing.T) {
		db, all := newDB(t)
		n, err := db.Trim(t.Context(), CheckID{Name: "dev1"}, TrimOptions{MaxCount: 2})
		if err != nil {
			t.Fatalf("Trim failed: %v", err)
		}
		if want, got := int64(nChecks*(nPer-2)), n; want != got {
			t.Fatalf("Trim removed %d records, want %d", got, want)
		}
		want := exclude(all, isDev(1), isRecordNumRange(0, 3))
		db.AssertRecords(want...)
	})
	t.Run("max all", func(t *testing.T) {
		db, all := newDB(t)
		n, err := db.Trim(t.Context(), CheckID{}, TrimOptions{MaxCount: 4})
		if err != nil {
			t.Fatalf("Trim failed: %v", err)
		}
		if want, got := int64(nDevices*nChecks*(nPer-4)), n; want != got {
			t.Fatalf("Trim removed %d records, want %d", got, want)
		}
		want := exclude(all, isRecordNumRange(0, 1))
		db.AssertRecords(want...)
	})
	t.Run("min>max", func(t *testing.T) {
		db, all := newDB(t)
		n, err := db.Trim(t.Context(), CheckID{}, TrimOptions{MinCount: 3, MaxCount: 2})
		if err != nil {
			t.Fatalf("Trim failed: %v", err)
		}
		if want, got := int64(nDevices*nChecks*(nPer-3)), n; want != got {
			t.Fatalf("Trim removed %d records, want %d", got, want)
		}
		want := exclude(all, isRecordNumRange(0, 2))
		db.AssertRecords(want...)
	})
	t.Run("large max", func(t *testing.T) {
		db, all := newDB(t)
		n, err := db.Trim(t.Context(), CheckID{}, TrimOptions{MaxCount: nPer*nDevices*nChecks + 1})
		if err != nil {
			t.Fatalf("Trim failed: %v", err)
		}
		if n != 0 {
			t.Fatalf("Trim removed %d records, want 0", n)
		}
		db.AssertRecords(all...)
	})

	t.Run("age name+id", func(t *testing.T) {
		db, all := newDB(t)
		before := dbEpoch.Add(createCadence * 8)
		n, err := db.Trim(t.Context(), CheckID{Name: "dev2", ID: "id1"}, TrimOptions{Before: before})
		if err != nil {
			t.Fatalf("Trim failed: %v", err)
		}
		if want, got := int64(8/(nDevices*nChecks)), n; want != got {
			t.Fatalf("Trim removed %d records, want %d", got, want)
		}
		want := exclude(all, isDev(2), isCheck(1), isBefore(before))
		db.AssertRecords(want...)
	})
	t.Run("age name", func(t *testing.T) {
		db, all := newDB(t)
		before := dbEpoch.Add(createCadence * 6)
		n, err := db.Trim(t.Context(), CheckID{Name: "dev1"}, TrimOptions{Before: before})
		if err != nil {
			t.Fatalf("Trim failed: %v", err)
		}
		if want, got := int64(6/nDevices), n; want != got {
			t.Fatalf("Trim removed %d records, want %d", got, want)
		}
		want := exclude(all, isDev(1), isBefore(before))
		db.AssertRecords(want...)
	})
	t.Run("age all", func(t *testing.T) {
		db, all := newDB(t)
		before := dbEpoch.Add(createCadence * 5)
		n, err := db.Trim(t.Context(), CheckID{}, TrimOptions{Before: before})
		if err != nil {
			t.Fatalf("Trim failed: %v", err)
		}
		if want, got := int64(5), n; want != got {
			t.Fatalf("Trim removed %d records, want %d", got, want)
		}
		want := exclude(all, isBefore(before))
		db.AssertRecords(want...)
	})

	// age+max, but age removes more
	t.Run("age<max", func(t *testing.T) {
		db, all := newDB(t)
		before := dbEpoch.Add(createCadence * nDevices * nChecks * 2)
		n, err := db.Trim(t.Context(), CheckID{}, TrimOptions{Before: before, MaxCount: 4})
		if err != nil {
			t.Fatalf("Trim failed: %v", err)
		}
		if want, got := int64(8), n; want != got {
			t.Fatalf("Trim removed %d records, want %d", got, want)
		}
		want := exclude(all, isBefore(before))
		db.AssertRecords(want...)
	})
	// age+max, but max removes more
	t.Run("age>max", func(t *testing.T) {
		db, all := newDB(t)
		before := dbEpoch.Add(createCadence * nDevices * nChecks * 2)
		n, err := db.Trim(t.Context(), CheckID{}, TrimOptions{Before: before, MaxCount: 2})
		if err != nil {
			t.Fatalf("Trim failed: %v", err)
		}
		if want, got := int64((nPer-2)*nDevices*nChecks), n; want != got {
			t.Fatalf("Trim removed %d records, want %d", got, want)
		}
		want := exclude(all, isRecordNumRange(0, nPer-2))
		db.AssertRecords(want...)
	})
	// age+min, keeps enough
	t.Run("age+min", func(t *testing.T) {
		db, all := newDB(t)
		before := dbEpoch.Add(createCadence * nDevices * nChecks * 4)
		n, err := db.Trim(t.Context(), CheckID{}, TrimOptions{Before: before, MinCount: 3})
		if err != nil {
			t.Fatalf("Trim failed: %v", err)
		}
		if want, got := int64(nDevices*nChecks*(nPer-3)), n; want != got {
			t.Fatalf("Trim removed %d records, want %d", got, want)
		}
		want := exclude(all, isRecordNumRange(0, nPer-3))
		db.AssertRecords(want...)
	})

	// tests with min+max+age
	t.Run("age<min<max", func(t *testing.T) {
		db, all := newDB(t)
		before := dbEpoch.Add(createCadence * nDevices * nChecks * 2)
		n, err := db.Trim(t.Context(), CheckID{}, TrimOptions{Before: before, MinCount: 2, MaxCount: 4})
		if err != nil {
			t.Fatalf("Trim failed: %v", err)
		}
		if want, got := int64(8), n; want != got {
			t.Fatalf("Trim removed %d records, want %d", got, want)
		}
		want := exclude(all, isBefore(before))
		db.AssertRecords(want...)
	})
	t.Run("min<age<max", func(t *testing.T) {
		db, all := newDB(t)
		before := dbEpoch.Add(createCadence * nDevices * nChecks * 3)
		n, err := db.Trim(t.Context(), CheckID{}, TrimOptions{Before: before, MinCount: 3, MaxCount: 4})
		if err != nil {
			t.Fatalf("Trim failed: %v", err)
		}
		if want, got := int64(nDevices*nChecks*(nPer-3)), n; want != got {
			t.Fatalf("Trim removed %d records, want %d", got, want)
		}
		want := exclude(all, isRecordNumRange(0, nPer-3))
		db.AssertRecords(want...)
	})
	t.Run("min<max<age", func(t *testing.T) {
		db, all := newDB(t)
		before := dbEpoch.Add(createCadence * nDevices * nChecks * 2)
		n, err := db.Trim(t.Context(), CheckID{}, TrimOptions{Before: before, MinCount: 2, MaxCount: 3})
		if err != nil {
			t.Fatalf("Trim failed: %v", err)
		}
		if want, got := int64(nDevices*nChecks*(nPer-3)), n; want != got {
			t.Fatalf("Trim removed %d records, want %d", got, want)
		}
		want := exclude(all, isRecordNumRange(0, nPer-3))
		db.AssertRecords(want...)
	})
	t.Run("age+min>max", func(t *testing.T) {
		db, all := newDB(t)
		before := dbEpoch.Add(createCadence * nDevices * nChecks * 3)
		n, err := db.Trim(t.Context(), CheckID{}, TrimOptions{Before: before, MinCount: 3, MaxCount: 2})
		if err != nil {
			t.Fatalf("Trim failed: %v", err)
		}
		if want, got := int64(nDevices*nChecks*(nPer-3)), n; want != got {
			t.Fatalf("Trim removed %d records, want %d", got, want)
		}
		want := exclude(all, isRecordNumRange(0, nPer-3))
		db.AssertRecords(want...)
	})
}

func TestDB_largeDB(t *testing.T) {
	if _, ok := os.LookupEnv("TEST_DB"); !ok {
		t.Skip("TEST_DB not set, skipping large database test")
	}
	db := newDBTester(t)
	const (
		nRecords      = 1_000_000
		batchSize     = 50000
		nNames        = 5000
		errMainRatio  = 5    // 1 in 5 will be an error check
		nAux          = 1000 // 1000 unique aux payloads
		logEveryBatch = 5
		countRuns     = 10
		readRuns      = 20
	)
	// device names
	names := func() []string {
		const prefix = "van/uk/brum/ugs/devices"
		devs := make([]string, nNames)
		for i := range nNames {
			devs[i] = fmt.Sprintf("%s/LTF-L01-%03d", prefix, i+1)
		}
		return devs
	}()
	checks := []string{"toner", "paper", "filter", "fuse", "filter", "waste", "paper", "filter"} // some duplicates to adjust frequency
	for i, check := range checks {
		checks[i] = fmt.Sprintf("smartcore.bos.autos.traitcheck:%s", check)
	}

	pbMarshal := func(m proto.Message) []byte {
		b, err := proto.Marshal(m)
		if err != nil {
			t.Fatalf("failed to marshal payload: %v", err)
		}
		return b
	}

	errMain := pbMarshal(&gen.HealthCheck{})
	valMain := pbMarshal(&gen.HealthCheck{
		Check: &gen.HealthCheck_Bounds_{Bounds: &gen.HealthCheck_Bounds{CurrentValue: &gen.HealthCheck_Value{Value: &gen.HealthCheck_Value_FloatValue{FloatValue: 24.76}}}},
	})
	main := func(i int) []byte {
		if i%errMainRatio == 0 {
			return errMain
		} else {
			return valMain
		}
	}
	sampleAux := pbMarshal(auxCheck())
	aux := func(i int) []byte {
		// This won't be a valid proto, but it will test different aux payloads.
		// Replace the first 4 bytes with the index mod nAux.
		binary.LittleEndian.PutUint32(sampleAux, uint32(i%nAux))
		return sampleAux
	}

	t.Logf("Inserting %d records in batches of %d...", nRecords, batchSize)
	insertStart := time.Now()
	batchStart := insertStart
	for batchNum := range nRecords / batchSize {
		batch := make([]Record, batchSize)
		for j := range batchSize {
			i := batchNum*batchSize + j
			batch[j] = db.NewRecord(names[i%nNames], checks[i%len(checks)], main(i), aux(i))
		}
		err := db.InsertBulk(t.Context(), batch)
		if err != nil {
			t.Fatalf("InsertBulk batch %d failed: %v", batchNum, err)
		}

		if b := batchNum + 1; b%logEveryBatch == 0 {
			rate := float64(b*batchSize) / time.Since(insertStart).Seconds()
			t.Logf("  Batch %d/%d inserted (%d/%d) in %v [%.2f/s]",
				b, nRecords/batchSize, b*batchSize, nRecords, time.Since(batchStart), rate)
			batchStart = time.Now()
		}
	}
	rate := nRecords / time.Since(insertStart).Seconds()
	t.Logf("Tnserted %d records in %v [%.2f/s]", nRecords, time.Since(insertStart), rate)

	// count timing
	n, err := db.Count(t.Context(), CheckID{}, 0, 0)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if n != nRecords {
		t.Fatalf("Count returned %d, want %d", n, nRecords)
	}

	countStart := time.Now()
	for range countRuns {
		_, err := db.Count(t.Context(), CheckID{}, 0, 0)
		if err != nil {
			t.Fatalf("Count failed: %v", err)
		}
	}
	countTotal := time.Since(countStart)
	t.Logf("All records counted in %v (%v runs, %v total)", countTotal/countRuns, countRuns, countTotal)

	// db sizing
	dbSize, err := db.Size(t.Context())
	if err != nil {
		t.Fatalf("Size failed: %v", err)
	}
	avgSizePerRecord := float64(dbSize) / nRecords

	// no splitting, no removal of ids, etc
	sampleProto := auxCheck()
	sampleProto.Id = "smartcore.bos.autos.traitcheck:toner"
	sampleProto.GetBounds().CurrentValue = &gen.HealthCheck_Value{Value: &gen.HealthCheck_Value_FloatValue{FloatValue: 24.76}}
	sampleProtoBytes := pbMarshal(sampleProto)
	samplePayloadSize := len(sampleProtoBytes) + // the proto payload
		len("van/uk/brum/ugs/devices/LTF-L01-001") + // the device name
		8 // an ID or timestamp
	cmpPercent := func(a, b float64) float64 {
		return (a - b) / b * 100
	}
	t.Logf("Protobuf size %.2f MB (%.2f bytes/record)", float64(nRecords*samplePayloadSize)/(1024*1024), float64(samplePayloadSize))
	t.Logf("Database size %.2f MB (%.2f bytes/record) [%+.2f%% vs proto]", float64(dbSize)/(1024*1024), avgSizePerRecord, cmpPercent(avgSizePerRecord, float64(samplePayloadSize)))

	// check read performance
	id := CheckID{Name: names[0], ID: checks[0]}
	dst := make([]Record, 100)
	from := MakeRecordID(db.epoch.Add(db.timeInc*1/8), 1)
	to := MakeRecordID(db.epoch.Add(db.timeInc*7/8), 10)
	n, err = db.Read(t.Context(), id, from, to, false, dst) // warmup
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	t.Logf("Running read test against %d records", n)
	readStart := time.Now()
	for range readRuns {
		n2, err := db.Read(t.Context(), id, from, to, false, dst) // warmup
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}
		if n2 != n {
			t.Fatalf("Read returned %d, want %d", n, n2)
		}
	}
	readTotal := time.Since(readStart)
	t.Logf("  Query read in %v (%v runs, %v total)", readTotal/readRuns, readRuns, readTotal)
	readStart = time.Now()
	for range readRuns {
		n2, err := db.Read(t.Context(), id, from, to, true, dst) // warmup
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}
		if n2 != n {
			t.Fatalf("Read returned %d, want %d", n, n2)
		}
	}
	readTotal = time.Since(readStart)
	t.Logf("  Desc read in %v (%v runs, %v total)", readTotal/readRuns, readRuns, readTotal)
}

func auxCheck() *gen.HealthCheck {
	return &gen.HealthCheck{
		DisplayName:    "A Bounds Check",
		Description:    "A description for a bounds check",
		OccupantImpact: gen.HealthCheck_COMFORT,
		Reliability:    &gen.HealthCheck_Reliability{State: gen.HealthCheck_Reliability_RELIABLE},
		Normality:      gen.HealthCheck_NORMAL,
		Check: &gen.HealthCheck_Bounds_{Bounds: &gen.HealthCheck_Bounds{
			Expected: &gen.HealthCheck_Bounds_NormalRange{NormalRange: &gen.HealthCheck_ValueRange{
				Low:      &gen.HealthCheck_Value{Value: &gen.HealthCheck_Value_FloatValue{FloatValue: 24.76}},
				High:     &gen.HealthCheck_Value{Value: &gen.HealthCheck_Value_FloatValue{FloatValue: 25.24}},
				Deadband: &gen.HealthCheck_Value{Value: &gen.HealthCheck_Value_FloatValue{FloatValue: 0.5}},
			}},
			DisplayUnit: "°C",
		}},
	}
}

var (
	// the earliest record in the test db has this create time
	dbEpoch       = time.Date(2021, 9, 4, 12, 0, 0, 0, time.UTC)
	createCadence = 10 * time.Minute // each record is this much later than the previous one
)

type dbTester struct {
	*testing.T
	*DB
	epoch   time.Time
	timeInc time.Duration
}

func newDBTester(t *testing.T) *dbTester {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	ctx := context.Background()
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	db, err := Open(ctx, dbPath, WithLogger(logger))
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	t.Logf("created test database %s", dbPath)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("failed to close test database: %v", err)
		}
		stat, err := os.Stat(dbPath)
		if err != nil {
			t.Logf("failed to stat test database file: %v", err)
		} else {
			t.Logf("database file size: %d bytes", stat.Size())
		}
	})
	return &dbTester{T: t, DB: db, epoch: dbEpoch}
}

func (t *dbTester) Run(name string, f func(t *dbTester)) {
	t.Helper()
	parent := t
	t.T.Run(name, func(t *testing.T) {
		f(&dbTester{T: t, DB: parent.DB, epoch: parent.epoch, timeInc: parent.timeInc})
	})
}

func (t *dbTester) NewRecord(name, id string, main, aux []byte) Record {
	t.Helper()
	return Record{
		Name:       name,
		CheckID:    id,
		CreateTime: t.tick(),
		Main:       main,
		Aux:        aux,
	}
}

func (t *dbTester) InsertRecord(name, id, main, aux string) Record {
	t.Helper()
	return t.InsertRecordBytes(name, id, []byte(main), []byte(aux))
}

func (t *dbTester) InsertRecordBytes(name, id string, main, aux []byte) Record {
	t.Helper()
	rec := t.NewRecord(name, id, main, aux)
	got, err := t.Insert(t.Context(), rec)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if err := t.testReadRecord(rec, got); err != nil {
		t.Fatalf("Insert returned invalid record: %v", err)
	}
	return got
}

func (t *dbTester) AssertRecords(want ...Record) {
	t.Helper()

	// space for more than we expect
	buf := make([]Record, len(want)+1)

	n, err := t.Read(t.Context(), CheckID{}, 0, 0, false, buf)
	t.AssertReadResponse(n, err, want, buf)
}

func (t *dbTester) AssertReadResponse(n int, err error, want, got []Record) {
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if n != len(want) {
		t.Fatalf("Read returned %d records, want %d", n, len(want))
	}
	var i int
	for i = 0; i < n; i++ {
		if err := t.testReadRecord(want[i], got[i]); err != nil {
			t.Errorf("Read record[%v]: %v", i, err)
		}
	}
	// remaining got records should be zero
	for ; i < len(got); i++ {
		if !got[i].IsZero() {
			t.Errorf("Read record[%v]: got %+v, want zero", i, got[i])
		}
	}
}

func (t *dbTester) AssertCount(want int) {
	t.Helper()
	n, err := t.Count(t.Context(), CheckID{}, 0, 0)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if n != want {
		t.Fatalf("Count returned %d, want %d", n, want)
	}
}

func (t *dbTester) testReadRecord(want, got Record) error {
	t.Helper()
	var err error
	if got.ID == 0 {
		err = errors.Join(err, fmt.Errorf("missing ID"))
	}
	if want.ID == 0 {
		got.ID = 0 // ignore ID in comparison
	}
	if diff := got.CreateTime.Sub(want.CreateTime).Abs(); diff > time.Millisecond {
		err = errors.Join(err, fmt.Errorf("create time diff %v > 1ms", diff))
	}
	// avoid time precision loss during comparison
	got.CreateTime = time.Time{}
	want.CreateTime = time.Time{}
	if diff := cmp.Diff(want, got); diff != "" {
		err = errors.Join(err, fmt.Errorf("record mismatch (-want +got):\n%s", diff))
	}
	return err
}

func (t *dbTester) tick() time.Time {
	now := t.epoch.Add(t.timeInc)
	t.timeInc += createCadence
	return now
}
