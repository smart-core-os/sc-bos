package udmi

import (
	"context"
	"strings"
	"testing"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/auth/policy"
	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
)

// fakeMessage is an mqtt.Message with a fixed topic and payload.
type fakeMessage struct {
	mqtt.Message
	topic   string
	payload []byte
}

func (m fakeMessage) Topic() string   { return m.topic }
func (m fakeMessage) Payload() []byte { return m.payload }

// fakeUdmiServer records OnMessage calls and returns err from each.
type fakeUdmiServer struct {
	udmipb.UnimplementedUdmiServiceServer
	err   error
	calls []*udmipb.OnMessageRequest
}

func (s *fakeUdmiServer) OnMessage(_ context.Context, req *udmipb.OnMessageRequest) (*udmipb.OnMessageResponse, error) {
	s.calls = append(s.calls, req)
	if s.err != nil {
		return nil, s.err
	}
	return &udmipb.OnMessageResponse{}, nil
}

func (s *fakeUdmiServer) client() udmipb.UdmiServiceClient {
	return udmipb.NewUdmiServiceClient(wrap.ServerToClient(udmipb.UdmiService_ServiceDesc, s))
}

// captureAuditor records every entry it is given.
type captureAuditor struct {
	entries []policy.IngressEntry
}

func (a *captureAuditor) AuditIngress(e policy.IngressEntry) {
	a.entries = append(a.entries, e)
}

// deliverMessages runs handleTopicChanges for a single set of topics, delivering payloads on each topic as soon as it
// is subscribed to.
func deliverMessages(t *testing.T, client udmipb.UdmiServiceClient, auditor *captureAuditor, payloads map[string][]string) {
	t.Helper()
	subscriber := SubscriberFunc(func(_ context.Context, topic string, cb mqtt.MessageHandler) error {
		for _, p := range payloads[topic] {
			cb(nil, fakeMessage{topic: topic, payload: []byte(p)})
		}
		return nil
	})
	changes := make(chan *udmipb.PullControlTopicsResponse, 1)
	var topics []string
	for topic := range payloads {
		topics = append(topics, topic)
	}
	changes <- &udmipb.PullControlTopicsResponse{Topics: topics}
	close(changes)

	var a auto.Auditor // keep a nil *captureAuditor out of the interface
	if auditor != nil {
		a = auditor
	}
	err := handleTopicChanges(context.Background(), "dev1", zap.NewNop(), client, changes, subscriber, a, "tcp://user:secret@broker:1883")
	if err != nil {
		t.Fatalf("handleTopicChanges: %v", err)
	}
}

func TestHandleTopicChanges_Audit(t *testing.T) {
	srv := &fakeUdmiServer{}
	auditor := &captureAuditor{}
	deliverMessages(t, srv.client(), auditor, map[string][]string{
		"site/dev1/config": {`{"a":1}`, `{"a":2}`},
	})

	if len(srv.calls) != 2 {
		t.Fatalf("OnMessage called %d times, want 2", len(srv.calls))
	}
	if len(auditor.entries) != 2 {
		t.Fatalf("got %d audit entries, want one per message (2)", len(auditor.entries))
	}
	e := auditor.entries[1]
	if e.Ingress != "mqtt" {
		t.Errorf("ingress = %q, want %q", e.Ingress, "mqtt")
	}
	if want := "tcp://user:xxxxx@broker:1883"; e.Peer != want {
		t.Errorf("peer = %q, want %q (password redacted)", e.Peer, want)
	}
	if e.Service != udmipb.UdmiService_ServiceDesc.ServiceName || e.Method != "OnMessage" {
		t.Errorf("service/method = %q/%q, want %q/OnMessage", e.Service, e.Method, udmipb.UdmiService_ServiceDesc.ServiceName)
	}
	if e.Err != nil {
		t.Errorf("err = %v, want nil", e.Err)
	}
	for key, want := range map[string]string{
		"target":      "dev1",
		"topic":       "site/dev1/config",
		"payload":     `{"a":2}`,
		"payloadSize": "7",
	} {
		if got := e.Fields[key]; got != want {
			t.Errorf("field %q = %q, want %q", key, got, want)
		}
	}
	if _, ok := e.Fields["payloadTruncated"]; ok {
		t.Errorf("payloadTruncated set on a short payload")
	}
}

// A message the driver rejects is still a write attempt and must be audited.
func TestHandleTopicChanges_AuditFailed(t *testing.T) {
	wantErr := status.Error(codes.InvalidArgument, "bad payload")
	srv := &fakeUdmiServer{err: wantErr}
	auditor := &captureAuditor{}
	deliverMessages(t, srv.client(), auditor, map[string][]string{
		"site/dev1/config": {`not json`},
	})

	if len(auditor.entries) != 1 {
		t.Fatalf("got %d audit entries, want 1", len(auditor.entries))
	}
	// compare by status, as the error has crossed the client/server boundary
	if got := status.Convert(auditor.entries[0].Err); got.Code() != codes.InvalidArgument || got.Message() != "bad payload" {
		t.Errorf("err = %v, want %v", auditor.entries[0].Err, wantErr)
	}
}

func TestHandleTopicChanges_AuditTruncatesPayload(t *testing.T) {
	payload := strings.Repeat("x", 2000)
	srv := &fakeUdmiServer{}
	auditor := &captureAuditor{}
	deliverMessages(t, srv.client(), auditor, map[string][]string{
		"site/dev1/config": {payload},
	})

	if len(auditor.entries) != 1 {
		t.Fatalf("got %d audit entries, want 1", len(auditor.entries))
	}
	f := auditor.entries[0].Fields
	if got := len(f["payload"]); got != maxAuditPayload {
		t.Errorf("audited payload is %d bytes, want %d", got, maxAuditPayload)
	}
	if f["payloadSize"] != "2000" {
		t.Errorf("payloadSize = %q, want %q", f["payloadSize"], "2000")
	}
	if f["payloadTruncated"] != "true" {
		t.Errorf("payloadTruncated = %q, want %q", f["payloadTruncated"], "true")
	}
	// the driver still gets the whole payload
	if got := srv.calls[0].GetMessage().GetPayload(); got != payload {
		t.Errorf("OnMessage payload was %d bytes, want %d", len(got), len(payload))
	}
}

func TestHandleTopicChanges_NilAuditor(t *testing.T) {
	srv := &fakeUdmiServer{}
	deliverMessages(t, srv.client(), nil, map[string][]string{
		"site/dev1/config": {`{}`},
	})
	if len(srv.calls) != 1 {
		t.Errorf("OnMessage called %d times, want 1", len(srv.calls))
	}
}

func TestTruncatePayload(t *testing.T) {
	tests := []struct {
		name          string
		in            string
		limit         int
		want          string
		wantTruncated bool
	}{
		{"shorter", "abc", 5, "abc", false},
		{"exact", "abcde", 5, "abcde", false},
		{"longer", "abcdef", 5, "abcde", true},
		// "é" is 2 bytes; cutting at 2 would split it, so it is dropped whole.
		{"rune boundary", "aé", 2, "a", true},
		{"multi-byte fits", "aé", 3, "aé", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, truncated := truncatePayload(tt.in, tt.limit)
			if got != tt.want || truncated != tt.wantTruncated {
				t.Errorf("truncatePayload(%q, %d) = %q, %v; want %q, %v", tt.in, tt.limit, got, truncated, tt.want, tt.wantTruncated)
			}
		})
	}
}

// Invalid UTF-8 can't be served over the LogApi, so it is replaced in the audit entry, while payloadSize still
// reports the original length.
func TestOnMessageAuditEntry_InvalidUTF8(t *testing.T) {
	e := onMessageAuditEntry("peer", "dev1", "topic", "a\xffb", nil)
	if got, want := e.Fields["payload"], "a\uFFFDb"; got != want {
		t.Errorf("payload = %q, want %q", got, want)
	}
	if got := e.Fields["payloadSize"]; got != "3" {
		t.Errorf("payloadSize = %q, want %q", got, "3")
	}
}
