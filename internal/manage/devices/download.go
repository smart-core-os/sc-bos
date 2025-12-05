package devices

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"fmt"
	"iter"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protopath"
	"google.golang.org/protobuf/reflect/protorange"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	timepb "github.com/smart-core-os/sc-api/go/types/time"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

//go:generate go tool protomod protoc -- -I . -I ../../../proto --go_out=paths=source_relative:. download.proto

type tokenClaims struct {
	Body string `json:"b"`
}

func (s *Server) GetDownloadDevicesUrl(_ context.Context, request *gen.GetDownloadDevicesUrlRequest) (*gen.DownloadDevicesUrl, error) {
	// validate
	switch request.MediaType {
	case "":
		request.MediaType = "text/csv"
	case "text/csv": // supported
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported media type %q", request.MediaType)
	}
	if err := validateQuery(request.Query); err != nil {
		return nil, err
	}
	if len(request.Filename) > 255 {
		return nil, status.Errorf(codes.InvalidArgument, "filename longer than 255")
	}

	expireAfter := s.now().Add(s.downloadExpiry)
	tokenStr, err := s.signAndSerializeDownloadToken(&DownloadToken{Request: request}, expireAfter)
	if err != nil {
		return nil, err
	}

	u := s.downloadUrlBase
	if err := s.writeDownloadToken(&u, tokenStr); err != nil {
		return nil, err
	}
	return &gen.DownloadDevicesUrl{
		Filename:        request.Filename,
		Url:             u.String(),
		MediaType:       request.MediaType,
		ExpireAfterTime: timestamppb.New(expireAfter),
	}, nil
}

// DownloadDevicesHTTPHandler responds to HTTP request urls returned by GetDownloadDevicesUrl.
// Requests must include a valid download token.
//
// CSV responses will include a header as the first row, for which columns are sorted alphabetically and grouped by md.name then md.* then *.
// For metadata columns each device is inspected to find all non-empty fields, each of which is included as a column in the md.* group.
// For trait values the supported traits and included columns is defined by the traitInfo map, see [Server.getTraitInfo] for details.
// Trait columns are fixed based on the advertised traits a device supports via its metadata.
// Typical column names are dot separated property paths, e.g. md.location.floor, access.grant, or meter.usage.
//
// Devices that have no traits or metadata (excluding their name and that they implement the metadata trait) are excluded from the response.
func (s *Server) DownloadDevicesHTTPHandler(w http.ResponseWriter, r *http.Request) {
	// 1. parse and validate the request
	tokenStr, err := s.readDownloadToken(r)
	if err != nil {
		http.Error(w, "invalid download token", http.StatusUnauthorized)
		return
	}

	writeErr := func(err error) {
		var httpErr httpError
		if errors.As(err, &httpErr) {
			http.Error(w, httpErr.msg, httpErr.code)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	token, err := s.parseAndValidateDownloadToken(tokenStr)
	if err != nil {
		writeErr(err)
		return
	}

	// 2. work out which devices to return and collect data headers
	traitInfo := s.getTraitInfo()
	deviceList, headers, err := s.listDevicesAndHeaders(token, traitInfo)
	if err != nil {
		writeErr(err)
		return
	}

	headerIndex := make(map[string]int)
	for i, h := range headers {
		headerIndex[h] = i
	}

	filename := token.Request.Filename
	if filename == "" {
		filename = "devices.csv"
	}

	// 3. start collecting the data and streaming it to the client
	// note: we only set the headers here (rather than earlier) to allow bad status codes to be returned if needed
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")

	// change the spelling of the headers if the user has asked us to
	renderedHeaders := headers
	if cols := token.GetRequest().GetTable().GetIncludeCols(); len(cols) > 0 {
		renderedHeaders = make([]string, 0, len(headers))
		titles := make(map[string]string, len(cols))
		for _, col := range cols {
			titles[col.Name] = col.Title
		}
		for _, h := range headers {
			if title, ok := titles[h]; ok {
				renderedHeaders = append(renderedHeaders, title)
				continue
			}
			renderedHeaders = append(renderedHeaders, h)
		}
	}

	csvOut := csv.NewWriter(w)
	if err := csvOut.Write(renderedHeaders); err != nil {
		return
	}
	out := newCSVWriter(csvOut, headerIndex)

	if token.Request.History == nil {
		s.writeLiveData(r.Context(), out, deviceList, traitInfo)
	} else {
		s.writeHistoricalData(r.Context(), out, deviceList, traitInfo, token.Request.History)
	}
	csvOut.Flush()
}

type writer interface {
	Write(row map[string]string)
	// HasAny returns true if any of the headers will be written by Write.
	HasAny(headers ...string) bool
}

func newCSVWriter(out *csv.Writer, headerIndex map[string]int) *csvWriter {
	return &csvWriter{out, headerIndex, make([]string, len(headerIndex))}
}

type csvWriter struct {
	out         *csv.Writer
	headerIndex map[string]int
	rowBuf      []string
}

func (c *csvWriter) Write(row map[string]string) {
	for h, v := range row {
		index, ok := c.headerIndex[h]
		if !ok {
			continue
		}
		c.rowBuf[index] = v
	}
	_ = c.out.Write(c.rowBuf)
	clear(c.rowBuf)
}

func (c *csvWriter) HasAny(headers ...string) bool {
	for _, h := range headers {
		if _, ok := c.headerIndex[h]; ok {
			return true
		}
	}
	return false
}

func (s *Server) writeLiveData(ctx context.Context, out writer, devices []*gen.Device, traitInfo map[string]traitInfo) {
	for _, d := range devices {
		row := make(map[string]string)
		captureMDValues(d, row)
		for _, t := range d.GetMetadata().GetTraits() {
			info, ok := traitInfo[t.Name]
			if !ok {
				continue
			}
			if !out.HasAny(info.headers...) {
				continue // skip get if the values won't be written
			}
			pageCtx, cleanup := context.WithTimeout(ctx, s.downloadPageTimeout)
			values, err := info.get(pageCtx, d.Name)
			cleanup()
			if err != nil {
				row[info.headers[0]] = fmt.Sprintf("ERR: %v", err)
				continue
			}
			maps.Copy(row, values)
		}
		out.Write(row)
	}
}

type source struct {
	device *gen.Device
	cursor *historyCursor
	skip   bool
}

func (s *Server) writeHistoricalData(ctx context.Context, out writer, devices []*gen.Device, traitInfo map[string]traitInfo, period *timepb.Period) {
	// we might want to allow the user to specify the order in the future
	const (
		// pageSize = 100
		// order    = "source"
		// pageSize = 10 // time order reads from all sources at once, limit memory use
		// order    = "time"
		pageSize = 100 // for device order we only keep the traits for a single device in memory at a time
		order    = "device"
	)

	var sources []*source
	for _, d := range devices {
		if d.Metadata == nil {
			continue
		}
		md := d.Metadata
		for _, t := range md.Traits {
			info, ok := traitInfo[t.Name]
			if !ok || info.history == nil {
				continue
			}
			if !out.HasAny(info.headers...) {
				continue // skip this source if none of the values will be written to the output
			}
			cursor := info.history(d.Name, period, pageSize)
			if cursor == nil {
				continue
			}
			sources = append(sources, &source{device: d, cursor: cursor})
		}
	}

	switch order {
	case "source":
		s.writeHistoryDataBySource(ctx, out, sources)
	case "device":
		s.writeHistoryDataByDevice(ctx, out, sources)
	case "time":
		s.writeHistoryDataByTime(ctx, out, sources)
	}
}

// writeHistoryDataBySource writes history data ordered by source, then time.
// This means each devices trait records are written together.
func (s *Server) writeHistoryDataBySource(ctx context.Context, out writer, sources []*source) {
	for _, source := range sources {
		mdVals := make(map[string]string)
		captureMDValues(source.device, mdVals)
		for {
			pageCtx, cleanup := context.WithTimeout(ctx, s.downloadPageTimeout)
			head, err := source.cursor.Head(pageCtx)
			cleanup()
			if err != nil {
				break
			}
			head.use()
			vals := head.vals
			maps.Copy(vals, mdVals)
			vals["timestamp"] = head.at.Format(time.DateTime)
			out.Write(vals)
		}
	}
}

// writeHistoryDataByDevice writes history data ordered by device, then time.
func (s *Server) writeHistoryDataByDevice(ctx context.Context, out writer, sources []*source) {
	sourcesByDevice := make(map[string][]*source)
	for _, source := range sources {
		sourcesByDevice[source.device.Name] = append(sourcesByDevice[source.device.Name], source)
	}

	for _, sources := range sourcesByDevice {
		s.writeHistoryDataByTime(ctx, out, sources)
	}
}

// writeHistoryDataByTime writes history data ordered by time.
// This means device trait records are interleaved in time order.
func (s *Server) writeHistoryDataByTime(ctx context.Context, out writer, sources []*source) {
	// cache of metadata values we can reuse during the main loop
	mds := make(map[string]map[string]string, len(sources))
	for _, source := range sources {
		if _, ok := mds[source.device.Name]; ok {
			continue
		}
		mdVals := make(map[string]string)
		captureMDValues(source.device, mdVals)
		mds[source.device.Name] = mdVals
	}

	for len(sources) > 0 {
		var (
			oldestRecord *historyRecord
			oldestSource *source
			anySkipped   bool
		)
		for _, source := range sources {
			pageCtx, cleanup := context.WithTimeout(ctx, s.downloadPageTimeout)
			head, err := source.cursor.Head(pageCtx)
			cleanup()
			if err != nil {
				anySkipped = true
				source.skip = true
				continue
			}
			switch {
			case oldestRecord == nil:
				oldestRecord, oldestSource = &head, source
			case head.at.Before(oldestRecord.at):
				oldestRecord, oldestSource = &head, source
			}
		}

		// both checks aren't strictly necessary as both are set at the same time,
		// however the static checker doesn't know that
		if oldestRecord == nil || oldestSource == nil {
			return // no records were processed
		}

		oldestRecord.use()
		vals := mds[oldestSource.device.Name]
		maps.Copy(vals, oldestRecord.vals)
		vals["timestamp"] = oldestRecord.at.Format(time.DateTime)
		out.Write(vals)

		if anySkipped {
			sources = slices.DeleteFunc(sources, func(source *source) bool {
				return source.skip
			})
		}
	}
}

func (s *Server) signAndSerializeDownloadToken(tokenBody *DownloadToken, expireAfter time.Time) (string, error) {
	tokenBodyStr, err := downloadTokenToString(tokenBody)
	if err != nil {
		return "", status.Errorf(codes.Unavailable, "token body creation error: %v", err)
	}

	key, err := s.downloadKey()
	if err != nil {
		return "", status.Errorf(codes.Unavailable, "token key creation error: %v", err)
	}

	// using JWT/JOSE here for the signing/key gen means we can also use it later for validation
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: key}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		return "", status.Errorf(codes.Unavailable, "token signer creation error: %v", err)
	}

	jwtClaims := &jwt.Claims{Expiry: jwt.NewNumericDate(expireAfter)}
	tokenClaims := tokenClaims{Body: tokenBodyStr}
	tokenStr, err := jwt.Signed(signer).Claims(tokenClaims).Claims(jwtClaims).Serialize()
	if err != nil {
		return "", status.Errorf(codes.Unavailable, "token serialization error: %v", err)
	}
	return tokenStr, nil
}

func (s *Server) parseAndValidateDownloadToken(tokenStr string) (*DownloadToken, error) {
	key, err := s.downloadKey()
	if err != nil {
		return nil, httpError{code: http.StatusInternalServerError, msg: "failed to read signing key"}
	}

	jwtToken, err := jwt.ParseSigned(tokenStr, []jose.SignatureAlgorithm{jose.HS256})
	if err != nil {
		return nil, httpError{code: http.StatusUnauthorized, msg: "invalid token"}
	}
	jwtClaims := &jwt.Claims{}
	var tokenClaims tokenClaims
	if err := jwtToken.Claims(key, jwtClaims, &tokenClaims); err != nil {
		return nil, httpError{code: http.StatusUnauthorized, msg: "untrusted token"}
	}
	if err := jwtClaims.ValidateWithLeeway(jwt.Expected{Time: s.now()}, s.downloadExpiryLeeway); err != nil {
		return nil, httpError{code: http.StatusUnauthorized, msg: "token expired"}
	}

	token, err := downloadTokenFromString(tokenClaims.Body)
	if err != nil {
		return nil, httpError{code: http.StatusUnauthorized, msg: "corrupted download token"}
	}
	return token, nil
}

func (s *Server) listDevicesAndHeaders(token *DownloadToken, traitInfo map[string]traitInfo) (devices []*gen.Device, headers []string, err error) {
	devices = s.m.ListDevices(
		resource.WithInclude(func(id string, item proto.Message) bool {
			if item == nil {
				return false
			}
			device := item.(*gen.Device)
			md := device.Metadata
			if len(md.GetTraits()) == 0 {
				return false
			}
			// Skip boring devices, aka those that have no metadata or other trait data.
			// They'd just show up as name=md.Name and a bunch of empty columns anyway.
			if proto.Equal(md, &traits.Metadata{Name: md.Name, Traits: []*traits.TraitMetadata{{Name: string(trait.Metadata)}}}) {
				return false
			}
			return deviceMatchesQuery(token.Request.Query, device)
		}),
	)

	tab := token.GetRequest().GetTable()

	// if the request specifies the included cols, use those instead of computing them from traits or metadata
	if len(tab.GetIncludeCols()) > 0 {
		headers := make([]string, 0, len(tab.GetIncludeCols()))
		for _, col := range tab.GetIncludeCols() {
			headers = append(headers, col.Name)
		}
		return devices, headers, nil
	}

	headerSet := make(map[string]struct{})
	if err := collectMetadataHeaders(headerSet, devices); err != nil {
		return nil, nil, err
	}

	collectTraitHeaders(headerSet, devices, traitInfo)

	if token.Request.History != nil {
		// delete headers for traits that don't support history
		for _, info := range traitInfo {
			if info.history == nil {
				for _, header := range info.headers {
					delete(headerSet, header)
				}
			}
		}
	}

	// delete any excluded headers
	for _, col := range tab.GetExcludeCols() {
		delete(headerSet, col.Name)
	}

	headers = sortHeaders(maps.Keys(headerSet))

	if token.Request.History != nil {
		// timestamp should be the first column
		headers = append([]string{"timestamp"}, headers...)
	}

	return devices, headers, nil
}

func collectMetadataHeaders(dst map[string]struct{}, deviceList []*gen.Device) error {
	// collect headers for all populated metadata fields
	for _, d := range deviceList {
		err := protorange.Range(d.ProtoReflect(), func(values protopath.Values) error {
			p := values.Path
			leafStep := p.Index(-1)
			switch leafStep.Kind() {
			case protopath.FieldAccessStep:
				fd := leafStep.FieldDescriptor()
				if fd == nil {
					return nil
				}
				if fd.Cardinality() == protoreflect.Repeated {
					return nil // no headers for lists
				}
				if fd.Kind() == protoreflect.MessageKind {
					return nil // no headers for messages
				}
			case protopath.ListIndexStep:
				fd := leafStep.FieldDescriptor()
				if fd == nil {
					return nil
				}
				if fd.Cardinality() == protoreflect.Repeated {
					return nil // no headers for lists
				}
				if fd.Kind() == protoreflect.MessageKind {
					return nil // no headers for messages
				}
			default:
			}

			header := protoPathToHeader(p)
			if header == "" {
				return nil
			}
			dst[header] = struct{}{}
			return nil
		})
		if err != nil {
			return httpError{http.StatusInternalServerError, "failed to collect headers"}
		}
	}
	// we process these separately
	delete(dst, "md.name")
	delete(dst, "md.traits.name")
	return nil
}

func collectTraitHeaders(dst map[string]struct{}, deviceList []*gen.Device, traitInfo map[string]traitInfo) {
	// capture trait headers
	traitNameSet := make(map[string]struct{})
	for _, d := range deviceList {
		if d.Metadata == nil {
			continue
		}
		for _, t := range d.Metadata.Traits {
			traitNameSet[t.Name] = struct{}{}
		}
	}

	for traitName := range traitNameSet {
		info, ok := traitInfo[traitName]
		if !ok {
			continue
		}
		for _, header := range info.headers {
			dst[header] = struct{}{}
		}
	}
}

func captureMDValues(device *gen.Device, row map[string]string) {
	_ = protorange.Range(device.ProtoReflect(), func(values protopath.Values) error {
		p := values.Path
		leafStep := p.Index(-1)
		switch leafStep.Kind() {
		case protopath.FieldAccessStep:
			fd := leafStep.FieldDescriptor()
			if fd == nil {
				return nil
			}
			if fd.Cardinality() == protoreflect.Repeated {
				return nil // no headers for lists
			}
			if fd.Kind() == protoreflect.MessageKind {
				return nil // no headers for messages
			}
		case protopath.ListIndexStep:
			fd := leafStep.FieldDescriptor()
			if fd == nil {
				return nil
			}
			if fd.Cardinality() == protoreflect.Repeated {
				return nil // no headers for lists
			}
			if fd.Kind() == protoreflect.MessageKind {
				return nil // no headers for messages
			}
		default:
		}

		header := protoPathToHeader(p)
		if header == "" {
			return nil
		}
		row[header] = values.Values[len(values.Values)-1].String()
		return nil
	})
}

func sortHeaders(headers iter.Seq[string]) []string {
	return slices.SortedFunc(headers, func(a string, b string) int {
		// sort name first
		switch {
		case a == "name" && b == "name":
			return 0
		case a == "name":
			return -1
		case b == "name":
			return 1
		}
		// sort metadata fields next
		aIsMD := strings.HasPrefix(a, "md.")
		bIsMD := strings.HasPrefix(b, "md.")
		switch {
		case aIsMD && !bIsMD:
			return -1
		case !aIsMD && bIsMD:
			return 1
		default:
			return strings.Compare(a, b)
		}
	})
}

type traitInfo struct {
	headers []string
	get     func(ctx context.Context, name string) (map[string]string, error)
	history func(name string, period *timepb.Period, pageSize int32) *historyCursor
}

func protoPathToHeader(p protopath.Path) string {
	var parts []string
	for _, step := range p {
		switch step.Kind() {
		case protopath.FieldAccessStep:
			parts = append(parts, string(step.FieldDescriptor().Name()))
		case protopath.MapIndexStep:
			parts = append(parts, step.MapIndex().String())
		case protopath.ListIndexStep:
			// skip writing the index, {bar: [{foo}]} -> bar.foo instead of bar[0].foo
		case protopath.RootStep:
			// skip writing the root
		case protopath.UnknownAccessStep:
		case protopath.AnyExpandStep:
		}
	}
	if len(parts) == 0 {
		return ""
	}
	if len(parts) >= 1 && parts[0] == "metadata" {
		parts[0] = "md" // shorten metadata to md as it's a common field
	}
	return strings.Join(parts, ".")
}

func (s *Server) RegisterHTTPMux(mux *http.ServeMux) {
	mux.Handle(s.downloadUrlBase.Path, gziphandler.GzipHandler(http.HandlerFunc(s.DownloadDevicesHTTPHandler)))
}

func newHMACKeyGen(size int) func() ([]byte, error) {
	// todo: support key rotation that doesn't invalidate unexpired tokens,
	//  I expect the sig will need to change to func(string)(string, []byte, error)
	key := make([]byte, size)
	var err error
	_, err = rand.Read(key)
	return func() ([]byte, error) {
		return key, err
	}
}

// DownloadTokenWriter is a function that writes a download token to a URL.
type DownloadTokenWriter = func(dst *url.URL, token string) error

// DownloadTokenReader is a function that reads a download token from a URL.
type DownloadTokenReader = func(*http.Request) (string, error)

func WithDownloadTokenCodec(w DownloadTokenWriter, r DownloadTokenReader) Option {
	return func(s *Server) {
		s.downloadTokenWriter = w
		s.downloadTokenReader = r
	}
}

func WithDownloadUrlBase(base url.URL) Option {
	return func(s *Server) {
		s.downloadUrlBase = base
	}
}

// readDownloadToken is the default implementation of DownloadTokenReader.
func readDownloadToken(r *http.Request) (string, error) {
	return r.URL.Query().Get("ddt"), nil
}

func (s *Server) readDownloadToken(r *http.Request) (string, error) {
	if s.downloadTokenReader == nil {
		return readDownloadToken(r)
	}
	return s.downloadTokenReader(r)
}

// writeDownloadToken is the default implementation of DownloadTokenWriter.
func writeDownloadToken(dst *url.URL, token string) error {
	q := dst.Query()
	q.Set("ddt", token)
	dst.RawQuery = q.Encode()
	return nil
}

func (s *Server) writeDownloadToken(dst *url.URL, token string) error {
	if s.downloadTokenWriter == nil {
		return writeDownloadToken(dst, token)
	}
	return s.downloadTokenWriter(dst, token)
}

func downloadTokenToString(token *DownloadToken) (string, error) {
	data, err := proto.Marshal(token)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func downloadTokenFromString(token string) (*DownloadToken, error) {
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}
	var dt DownloadToken
	if err := proto.Unmarshal(data, &dt); err != nil {
		return nil, err
	}
	return &dt, nil
}

type httpError struct {
	code int
	msg  string
}

func (h httpError) Error() string {
	return http.StatusText(h.code) + ": " + h.msg
}
