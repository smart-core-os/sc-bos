package test

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	reflectionpb "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/internal/util/grpc/reflectionapi"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/hubpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/logpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/onoffpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/servicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/system/gateway/internal/test/shared"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

var skipBuild = flag.Bool("skip-build", false, "skip building and running binaries")
var ignoreEnrolErr = flag.Bool("ignore-enrol-err", false, "ignore enrolment errors")

// TestGateway_e2e tests the gateway by running a cohort of nodes, each a different sc-bos process.
// The test only runs if the -short flag is not set.
func TestGateway_e2e(t *testing.T) {
	// WARNING: This test doesn't play perfectly with go tests caching for a number of reasons:
	// 1. it builds go binaries which are the target of the tests which don't get checked as part of cache invalidation
	// 2. those binaries read files that are also not part of the cache invalidation

	if testing.Short() {
		t.Skip("long test")
	}

	ctx := t.Context()

	if !*skipBuild {
		runNodes(t, ctx)
	}

	// Next up we need to configure the cohort
	t.Logf("Configuring cohort")
	cohortStart := time.Now()
	configureCohort(t, ctx)
	t.Logf("Cohort configured in %s", time.Since(cohortStart))

	// Finally we're ready to start checking the setup
	for i, addr := range shared.GWGRPCAddrs {
		t.Run(fmt.Sprintf("gw%d %s", i+1, addr), func(t *testing.T) {
			// this timeout is long because the GW is using an exponential backoff for retries,
			// capped at 30s, but all attempts before the cohort is configured increase the delay.
			testCtx, stopTests := context.WithTimeout(ctx, 60*time.Second)
			defer stopTests()
			// these func log themselves
			testGW(t, testCtx, addr)
		})
	}
}

func runNodes(t *testing.T, ctx context.Context) {
	t.Helper()
	// We can't use `go run`, even though it has better cache semantics,
	// because sending kill/interrupt to the `go run` process does not forward
	// those signals to the bos process which causes them to hang the test process.
	dir := t.TempDir()

	t.Logf("Building binaries")
	buildStart := time.Now()
	buildAll(t, dir)

	t.Logf("Running nodes")
	runStart := time.Now()
	go runAllNodes(t, ctx, dir)

	// Wait for the nodes to start up, shouldn't take more than a few seconds on _decent_ hardware.
	startCtx, cancelStart := context.WithTimeout(ctx, 30*time.Second)
	defer cancelStart()
	waitForNodes(t, startCtx)
	t.Logf("All nodes running in %s (b=%s,w=%s)", time.Since(buildStart), runStart.Sub(buildStart), time.Since(runStart))
}

func buildAll(t *testing.T, dir string) {
	t.Helper()

	ctx, stop := newCtx(t)
	defer stop()
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return build(t, ctx, "gw", dir) })
	g.Go(func() error { return build(t, ctx, "ac", dir) })
	g.Go(func() error { return build(t, ctx, "hub", dir) })
	if err := g.Wait(); err != nil {
		t.Fatal("build failed", err)
	}
}

func build(t *testing.T, ctx context.Context, name, dir string) error {
	t.Helper()

	build := exec.CommandContext(ctx, "go", "build", "-o", filepath.Join(dir, name), "./"+name+"/cmd")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	return build.Run()
}

func runAllNodes(t *testing.T, ctx context.Context, dir string) {
	t.Helper()

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return runNode(t, ctx, dir, "hub", shared.HubGRPCAddr, shared.HubHTTPSAddr) })
	for i, addrs := range zip(shared.ACGRPCAddrs, shared.ACHTTPSAddrs) {
		g.Go(func() error { return runNode(t, ctx, dir, fmt.Sprintf("ac%d", i+1), addrs[0], addrs[1]) })
	}
	for i, addrs := range zip(shared.GWGRPCAddrs, shared.GWHTTPSAddrs) {
		g.Go(func() error { return runNode(t, ctx, dir, fmt.Sprintf("gw%d", i+1), addrs[0], addrs[1]) })
	}
	if err := g.Wait(); err != nil {
		select {
		case <-ctx.Done():
			return
		default:
		}
		t.Error("run failed", err)
	}
}

func runNode(t *testing.T, ctx context.Context, dir, name, grpcAddr, httpsAddr string) error {
	t.Helper()

	execName := strings.TrimRight(name, "1234567890")

	node := exec.CommandContext(ctx, filepath.Join(dir, execName),
		"--listen-grpc", grpcAddr,
		"--listen-https", httpsAddr,
		"--policy-mode", "off", // disable policy checking for now
		"--sysconf", filepath.Join("testdata", name, "system.conf.json"),
		"--appconf", filepath.Join("testdata", name, "app.conf.json"),
		"--data", filepath.Join(t.TempDir(), name+"-data"),
	)
	node.Stdout = os.Stdout
	node.Stderr = os.Stderr
	return node.Run()
}

func waitForNodes(t *testing.T, ctx context.Context) {
	t.Helper()

	waitForNode(t, ctx, shared.HubGRPCAddr)
	for _, addr := range shared.ACGRPCAddrs {
		waitForNode(t, ctx, addr)
	}
	for _, addr := range shared.GWGRPCAddrs {
		waitForNode(t, ctx, addr)
	}
}

func waitForNode(t *testing.T, ctx context.Context, addr string) {
	t.Helper()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: true,
	})))
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer conn.Close()

	client := metadatapb.NewMetadataApiClient(conn)
	err = backoff.Retry(func() error {
		_, err := client.GetMetadata(ctx, &metadatapb.GetMetadataRequest{})
		if code := status.Code(err); err != nil && code != codes.Unavailable {
			t.Logf("failed to poll node %q for liveness: %v", addr, err)
		}
		return err
	}, backoff.WithContext(backoff.NewExponentialBackOff(), ctx))
	if err != nil {
		t.Fatalf("wait for node %s: %v", addr, err)
	}
}

func configureCohort(t *testing.T, ctx context.Context) {
	t.Helper()

	// todo: use the hubs ca (should be in dir, after the first request) for our client cert checks

	hubConn, err := grpc.NewClient(shared.HubGRPCAddr, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: true,
	})))
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer hubConn.Close()

	checkErr := func(addr string, err error) {
		t.Helper()
		if *ignoreEnrolErr {
			return
		}
		if err != nil {
			t.Fatalf("enroll %s: %v", addr, err)
		}
	}

	client := hubpb.NewHubApiClient(hubConn)
	for _, addr := range shared.ACGRPCAddrs {
		_, err := client.EnrollHubNode(ctx, &hubpb.EnrollHubNodeRequest{Node: &hubpb.HubNode{Address: addr}})
		checkErr(addr, err)
	}
	for _, addr := range shared.GWGRPCAddrs {
		_, err := client.EnrollHubNode(ctx, &hubpb.EnrollHubNodeRequest{Node: &hubpb.HubNode{Address: addr}})
		checkErr(addr, err)
	}
}

func testGW(t *testing.T, ctx context.Context, addr string) {
	t.Helper()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: true,
	})))
	if err != nil {
		t.Fatalf("dial %s: %v", addr, err)
	}
	defer conn.Close()

	// Named devices are correctly routed
	nodeDevices := []string{
		"hub", "ac1", "ac2",
		// one of these will be the node we're talking to, but either way it should exist
		"gw1", "gw2",
	}
	onOffDevices := []string{
		"ac1/dev1",
		"ac2/dev1",
		"hub/dev1",
	}
	serviceDevices := []string{"automations", "drivers", "systems", "zones"}
	for _, node := range nodeDevices {
		for _, s := range []string{"automations", "drivers", "systems", "zones"} {
			serviceDevices = append(serviceDevices, fmt.Sprintf("%s/%s", node, s))
		}
	}

	// these tests are mostly about waiting for the gw to finish its setup
	t.Run("node devices online", func(t *testing.T) {
		for _, name := range nodeDevices {
			waitForDevice(t, ctx, conn, name)
		}
	})
	t.Run("onOff devices online", func(t *testing.T) {
		for _, name := range onOffDevices {
			waitForDevice(t, ctx, conn, name)
		}
	})
	t.Run("service devices online", func(t *testing.T) {
		for _, name := range serviceDevices {
			waitForDevice(t, ctx, conn, name)
		}
	})

	// tests that devices appear in gw DevicesApi responses
	t.Run("devices api includes devices", func(t *testing.T) {
		client := devicespb.NewDevicesApiClient(conn)
		testDevicesApiHasNames(t, ctx, addr, onOffDevices, client, &devicespb.ListDevicesRequest{
			Query: &devicespb.Device_Query{Conditions: []*devicespb.Device_Query_Condition{
				{Field: "metadata.traits.name", Value: &devicespb.Device_Query_Condition_StringEqual{StringEqual: string(trait.OnOff)}},
			}},
		})
	})

	t.Run("onOff devices respond", func(t *testing.T) {
		client := onoffpb.NewOnOffApiClient(conn)
		for _, name := range onOffDevices {
			testOnOffApi(t, ctx, addr, name, client)
		}
	})
	t.Run("services respond", func(t *testing.T) {
		client := servicespb.NewServicesApiClient(conn)
		for _, name := range serviceDevices {
			testServicesApi(t, ctx, addr, name, client)
		}
	})
	t.Run("has health check", func(t *testing.T) {
		client := healthpb.NewHealthApiClient(conn)
		devicesClient := devicespb.NewDevicesApiClient(conn)
		for _, name := range onOffDevices {
			testHealthApi(t, ctx, addr, name, client)
			testHealthCheckIdsMatch(t, ctx, addr, name, client, devicesClient)
		}
	})
	t.Run("devices api has health checks", func(t *testing.T) {
		client := devicespb.NewDevicesApiClient(conn)
		testDevicesApiHasNames(t, ctx, addr, onOffDevices, client, &devicespb.ListDevicesRequest{
			Query: &devicespb.Device_Query{Conditions: []*devicespb.Device_Query_Condition{
				{Field: "health_checks.bounds.normal_value.string_value", Value: &devicespb.Device_Query_Condition_StringEqual{StringEqual: "ON"}},
			}},
		})
	})

	t.Run("reflection", func(t *testing.T) {
		testReflection(t, ctx, conn)
	})

	t.Run("stable device list", func(t *testing.T) {
		testStableDeviceList(t, ctx, conn)
	})

	zoneDevices := []string{
		"ac1/zone1",
		"ac2/zone1",
	}
	t.Run("zone devices online", func(t *testing.T) {
		for _, name := range zoneDevices {
			waitForDevice(t, ctx, conn, name)
		}
	})
	t.Run("zone services list zones", func(t *testing.T) {
		client := servicespb.NewServicesApiClient(conn)
		testZoneServicesApi(t, ctx, addr, "ac1/zones", "ac1/zone1", client)
		testZoneServicesApi(t, ctx, addr, "ac2/zones", "ac2/zone1", client)
	})

	t.Run("aggregated logs", func(t *testing.T) {
		testLogAggregation(t, ctx, conn)
	})

	testHubApis(t, ctx, conn)
}

// testLogAggregation checks that PullLogMessages against the gateway's aggregate
// "logs" name merges logs from more than one cohort node, each tagged with its
// source. ac1 and ac2 run the log system; their startup logs are captured and
// should surface here.
func testLogAggregation(t *testing.T, ctx context.Context, conn *grpc.ClientConn) {
	t.Helper()

	client := logpb.NewLogApiClient(conn)
	want := map[string]bool{"ac1": false, "ac2": false}

	// The gateway may still be discovering the ACs' log devices, so retry until
	// both sources appear (or ctx is done). Each attempt reads the initial replay,
	// which is bounded, then stops.
	err := backoff.Retry(func() error {
		attemptCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		stream, err := client.PullLogMessages(attemptCtx, &logpb.PullLogMessagesRequest{Name: "logs", InitialCount: 200})
		if err != nil {
			return err
		}
		for {
			resp, err := stream.Recv()
			if err != nil {
				break // attempt timed out or stream ended; assess what we collected
			}
			for _, change := range resp.Changes {
				for _, m := range change.Messages {
					if m.Source == "" {
						t.Errorf("aggregated log message has empty source: %q", m.Message)
					}
					if _, ok := want[m.Source]; ok {
						want[m.Source] = true
					}
				}
			}
			if want["ac1"] && want["ac2"] {
				return nil
			}
		}
		var missing []string
		for src, seen := range want {
			if !seen {
				missing = append(missing, src)
			}
		}
		slices.Sort(missing)
		return fmt.Errorf("no aggregated logs yet from sources: %v", missing)
	}, backoff.WithContext(backoff.NewExponentialBackOff(backoff.WithMaxInterval(2*time.Second)), ctx))
	if err != nil {
		t.Fatalf("aggregated logs: %v", err)
	}
}

func waitForDevice(t *testing.T, ctx context.Context, conn *grpc.ClientConn, name string) {
	t.Helper()

	client := metadatapb.NewMetadataApiClient(conn)
	err := backoff.Retry(func() error {
		_, err := client.GetMetadata(ctx, &metadatapb.GetMetadataRequest{Name: name})
		return err
	}, backoff.WithContext(backoff.NewExponentialBackOff(backoff.WithMaxInterval(5*time.Second)), ctx))
	if err != nil {
		t.Fatalf("wait for device %s: %v", name, err)
	}
}

func testDevicesApiHasNames(t *testing.T, ctx context.Context, addr string, names []string, client devicespb.DevicesApiClient, request *devicespb.ListDevicesRequest) {
	t.Helper()

	res, err := client.ListDevices(ctx, request)
	if err != nil {
		t.Fatalf("[%s] list devices: %v", addr, err)
	}
	gotNames := make([]string, len(res.Devices))
	for i, d := range res.Devices {
		gotNames[i] = d.Name
	}
	sortStrings := cmpopts.SortSlices(func(a, b string) bool { return a < b })
	if diff := cmp.Diff(names, gotNames, sortStrings); diff != "" {
		t.Fatalf("[%s] list devices: unexpected response (-want +got):\n%s", addr, diff)
	}
}

func testOnOffApi(t *testing.T, ctx context.Context, addr, name string, client onoffpb.OnOffApiClient) {
	t.Helper()

	// useful for cancelling the stream
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// set initial known state: ON
	res, err := client.UpdateOnOff(ctx, &onoffpb.UpdateOnOffRequest{Name: name, OnOff: &onoffpb.OnOff{State: onoffpb.OnOff_ON}})
	if err != nil {
		t.Fatalf("[%s] update onoff %s: %v", addr, name, err)
	}
	if diff := cmp.Diff(&onoffpb.OnOff{State: onoffpb.OnOff_ON}, res, protocmp.Transform()); diff != "" {
		t.Fatalf("[%s] update onoff %s: unexpected response (-want +got):\n%s", addr, name, diff)
	}

	// subscribe
	changes := make(chan *onoffpb.PullOnOffResponse, 1) // we're only expecting 1
	stream, err := client.PullOnOff(ctx, &onoffpb.PullOnOffRequest{Name: name, UpdatesOnly: true})
	if err != nil {
		t.Fatalf("[%s] pull onoff %s: %v", addr, name, err)
	}
	go func() {
		for {
			res, err := stream.Recv()
			if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled {
				close(changes)
				return
			}
			if err != nil {
				t.Errorf("[%s] pull onoff %s: %v", addr, name, err)
				return
			}
			changes <- res
		}
	}()

	// check initial state
	res, err = client.GetOnOff(ctx, &onoffpb.GetOnOffRequest{Name: name})
	if err != nil {
		t.Fatalf("[%s] get onoff %s: %v", addr, name, err)
	}
	if diff := cmp.Diff(&onoffpb.OnOff{State: onoffpb.OnOff_ON}, res, protocmp.Transform()); diff != "" {
		t.Fatalf("[%s] get onoff %s: unexpected response (-want +got):\n%s", addr, name, diff)
	}

	// Perform update and check for change
	res, err = client.UpdateOnOff(ctx, &onoffpb.UpdateOnOffRequest{Name: name, OnOff: &onoffpb.OnOff{State: onoffpb.OnOff_OFF}})
	if err != nil {
		t.Fatalf("[%s] update onoff %s: %v", addr, name, err)
	}
	if diff := cmp.Diff(&onoffpb.OnOff{State: onoffpb.OnOff_OFF}, res, protocmp.Transform()); diff != "" {
		t.Fatalf("[%s] update onoff %s: unexpected response (-want +got):\n%s", addr, name, diff)
	}
	pullRes := <-changes
	want := &onoffpb.PullOnOffResponse{Changes: []*onoffpb.PullOnOffResponse_Change{
		{
			Name:  name,
			OnOff: &onoffpb.OnOff{State: onoffpb.OnOff_OFF},
		},
	}}
	// clear timestamps to make comparing easier
	for i := range pullRes.Changes {
		pullRes.Changes[i].ChangeTime = nil
	}
	if diff := cmp.Diff(want, pullRes, protocmp.Transform()); diff != "" {
		t.Fatalf("[%s] pull onoff %s: unexpected response (-want +got):\n%s", addr, name, diff)
	}
}

func testServicesApi(t *testing.T, ctx context.Context, addr, name string, client servicespb.ServicesApiClient) {
	t.Helper()

	_, err := client.ListServices(ctx, &servicespb.ListServicesRequest{Name: name})
	if err != nil {
		t.Fatalf("[%s] list services %s: %v", addr, name, err)
	}
}

func testZoneServicesApi(t *testing.T, ctx context.Context, addr, name, wantZone string, client servicespb.ServicesApiClient) {
	t.Helper()
	res, err := client.ListServices(ctx, &servicespb.ListServicesRequest{Name: name})
	if err != nil {
		t.Fatalf("[%s] list services %s: %v", addr, name, err)
	}
	for _, svc := range res.Services {
		if svc.Id == wantZone {
			return
		}
	}
	t.Fatalf("[%s] list services %s: zone %q not found in response (got %d services)", addr, name, wantZone, len(res.Services))
}

func testHealthApi(t *testing.T, ctx context.Context, addr, name string, client healthpb.HealthApiClient) {
	t.Helper()
	res, err := client.ListHealthChecks(ctx, &healthpb.ListHealthChecksRequest{Name: name})
	if err != nil {
		t.Fatalf("[%s] list health checks %s: %v", addr, name, err)
	}
	if len(res.HealthChecks) == 0 {
		t.Fatalf("[%s] list health checks %s: no health checks found", addr, name)
	}
	// the checks we've set up are looking for OnOff being ON,
	// make sure they exist and the checks we see aren't just some defaults
	foundOnOffCheck := false
	for _, check := range res.HealthChecks {
		if want := check.GetBounds().GetNormalValue().GetStringValue(); want == "ON" {
			foundOnOffCheck = true
			// the measured value (current_value) is set by the healthbounds auto on the AC.
			// it must survive proxying through the gateway, otherwise the ops UI can't show it.
			if cv := check.GetBounds().GetCurrentValue(); cv == nil {
				t.Errorf("[%s] list health checks %s: OnOff check has no current_value (measured value lost through gateway)", addr, name)
			}
			break
		}
	}
	if !foundOnOffCheck {
		t.Fatalf("[%s] list health checks %s: no OnOff=ON health check found", addr, name)
	}

	// The ops UI uses PullHealthChecks (streaming), not ListHealthChecks, to read the
	// measured value. Verify the current_value also survives the streaming proxy.
	pullCtx, stopPull := context.WithTimeout(ctx, 10*time.Second)
	defer stopPull()
	stream, err := client.PullHealthChecks(pullCtx, &healthpb.PullHealthChecksRequest{Name: name})
	if err != nil {
		t.Fatalf("[%s] pull health checks %s: %v", addr, name, err)
	}
	pullFoundOnOff, pullHasCurrentValue := false, false
	for !pullFoundOnOff {
		msg, err := stream.Recv()
		if err != nil {
			break
		}
		for _, change := range msg.GetChanges() {
			check := change.GetNewValue()
			if check.GetBounds().GetNormalValue().GetStringValue() != "ON" {
				continue
			}
			pullFoundOnOff = true
			pullHasCurrentValue = check.GetBounds().GetCurrentValue() != nil
			break
		}
	}
	if !pullFoundOnOff {
		t.Errorf("[%s] pull health checks %s: no OnOff=ON health check found via pull", addr, name)
	} else if !pullHasCurrentValue {
		t.Errorf("[%s] pull health checks %s: OnOff check has no current_value via pull (measured value lost through gateway)", addr, name)
	}
}

// testHealthCheckIdsMatch verifies that the check ids reported by the DevicesApi (the device list,
// which has measured values stripped) match the ids reported by the HealthApi (which carries the
// measured values). The ops UI merges the two by id to display measured values in the expanded row,
// so a mismatch through the gateway would mean the measured value never shows.
func testHealthCheckIdsMatch(t *testing.T, ctx context.Context, addr, name string, healthClient healthpb.HealthApiClient, devicesClient devicespb.DevicesApiClient) {
	t.Helper()

	healthRes, err := healthClient.ListHealthChecks(ctx, &healthpb.ListHealthChecksRequest{Name: name})
	if err != nil {
		t.Fatalf("[%s] list health checks %s: %v", addr, name, err)
	}
	healthIds := make(map[string]struct{})
	for _, c := range healthRes.GetHealthChecks() {
		healthIds[c.GetId()] = struct{}{}
	}

	devRes, err := devicesClient.ListDevices(ctx, &devicespb.ListDevicesRequest{
		Query: &devicespb.Device_Query{Conditions: []*devicespb.Device_Query_Condition{
			{Field: "name", Value: &devicespb.Device_Query_Condition_StringEqual{StringEqual: name}},
		}},
	})
	if err != nil {
		t.Fatalf("[%s] list devices %s: %v", addr, name, err)
	}
	if len(devRes.GetDevices()) != 1 {
		t.Fatalf("[%s] list devices %s: expected 1 device, got %d", addr, name, len(devRes.GetDevices()))
	}
	for _, c := range devRes.GetDevices()[0].GetHealthChecks() {
		if c.GetId() == "" {
			t.Errorf("[%s] device %s: DevicesApi health check has empty id (ops UI can't merge in measured value)", addr, name)
			continue
		}
		if _, ok := healthIds[c.GetId()]; !ok {
			t.Errorf("[%s] device %s: DevicesApi check id %q not found in HealthApi ids %v (id mismatch through gateway)", addr, name, c.GetId(), healthIds)
		}
	}
}

func testReflection(t *testing.T, ctx context.Context, conn *grpc.ClientConn) {
	ctx, stop := context.WithCancel(ctx)
	defer stop()

	client := reflectionpb.NewServerReflectionClient(conn)
	stream, err := client.ServerReflectionInfo(ctx)
	if err != nil {
		t.Fatal("server reflection info:", err)
	}

	services, err := reflectionapi.ListServices(stream)
	if err != nil {
		t.Fatal("list services:", err)
	}
	wantServices := []*reflectionpb.ServiceResponse{
		{Name: "grpc.reflection.v1.ServerReflection"},
		{Name: "grpc.reflection.v1alpha.ServerReflection"},
		{Name: "smartcore.bos.devices.v1.DevicesApi"},
		{Name: "smartcore.bos.enrollment.v1.EnrollmentApi"},
		{Name: "smartcore.bos.health.v1.HealthApi"},
		{Name: "smartcore.bos.health.v1.HealthHistory"},
		{Name: "smartcore.bos.hub.v1.HubApi"},
		{Name: "smartcore.bos.log.v1.LogApi"},
		{Name: "smartcore.bos.metadata.v1.MetadataApi"},
		{Name: "smartcore.bos.mock.v1.MockDeviceApi"},
		{Name: "smartcore.bos.onoff.v1.OnOffApi"},
		{Name: "smartcore.bos.onoff.v1.OnOffInfo"},
		{Name: "smartcore.bos.parent.v1.ParentApi"},
		{Name: "smartcore.bos.services.v1.ServicesApi"},
	}
	services = slices.DeleteFunc(services, func(response *reflectionpb.ServiceResponse) bool {
		// ignore private BOS APIs, as these are expected to change fairly often
		return strings.HasPrefix(response.Name, "smartcore.bos.ops.")
	})
	if diff := cmp.Diff(wantServices, services, protocmp.Transform()); diff != "" {
		t.Fatalf("services: (-want +got):\n%s", diff)
	}

	types := []string{
		"smartcore.bos.onoff.v1.OnOffApi",
		"smartcore.bos.devices.v1.DevicesApi",
	}
	for _, typ := range types {
		_, err = reflectionapi.FileContainingSymbol(stream, typ)
		if err != nil {
			t.Fatalf("file containing symbol %s: %v", typ, err)
		}
	}

	unknownTypes := []string{
		"smartcore.traits.UnknownApiForTesting", // doesn't exist
		// note, all apis that are in the traits or gen packages get loaded at the same time (during package init)
	}
	for _, typ := range unknownTypes {
		_, err = reflectionapi.FileContainingSymbol(stream, typ)
		if status.Code(err) != codes.NotFound {
			t.Fatalf("file containing symbol %s: expected error, got %v", typ, err)
		}
	}
}

func testStableDeviceList(t *testing.T, ctx context.Context, conn *grpc.ClientConn) {
	ctx, stop := context.WithTimeout(ctx, 2*time.Second)
	defer stop()
	client := devicespb.NewDevicesApiClient(conn)
	type totals struct{ add, update, remove, replace int }
	// used to track what is being unstable,
	// technically we could fail on the first event,
	// but this way gives more info about what is unstable.
	events := make(map[string]totals)
	// this stream shouldn't receive anything
	openStream := func() grpc.ServerStreamingClient[devicespb.PullDevicesResponse] {
		stream, err := client.PullDevices(ctx, &devicespb.PullDevicesRequest{UpdatesOnly: true})
		if code := status.Code(err); code == codes.DeadlineExceeded {
			return nil
		}
		if err != nil {
			t.Fatalf("pull devices: %v", err)
		}
		return stream
	}
	stream := openStream()

	for {
		res, err := stream.Recv()
		if code := status.Code(err); code == codes.DeadlineExceeded {
			break // our timeout has elapsed
		}
		if errors.Is(err, io.EOF) {
			t.Logf("pull devices stream closed by server (EOF), reopening")
			stream = openStream()
			if stream == nil {
				break
			}
			continue
		}
		if err != nil {
			t.Fatalf("recv pull devices: %v", err)
		}
		for _, change := range res.Changes {
			total := events[change.Name]
			switch change.Type {
			case typespb.ChangeType_ADD:
				total.add++
			case typespb.ChangeType_UPDATE:
				total.update++
			case typespb.ChangeType_REMOVE:
				total.remove++
			case typespb.ChangeType_REPLACE:
				total.replace++
			default:
				t.Fatalf("unknown change type: %v", change.Type)
			}
			events[change.Name] = total
		}
	}

	if len(events) > 0 {
		var sb strings.Builder
		sb.WriteString("device list unstable, received events:\n")
		for name, total := range events {
			fmt.Fprintf(&sb, "\t%s: %+v\n", name, total)
		}
		t.Fatal(sb.String())
	}
}

func testHubApis(t *testing.T, ctx context.Context, conn *grpc.ClientConn) {
	t.Helper()

	t.Run("HubApi", func(t *testing.T) {
		client := hubpb.NewHubApiClient(conn)
		res, err := client.ListHubNodes(ctx, &hubpb.ListHubNodesRequest{})
		if err != nil {
			t.Fatalf("list hub nodes: %v", err)
		}
		wantNames := []string{"ac1", "ac2", "gw1", "gw2"}
		gotNames := make([]string, len(res.Nodes))
		for i, node := range res.Nodes {
			gotNames[i] = node.Name
		}
		sortStrings := cmpopts.SortSlices(func(a, b string) bool { return a < b })
		if diff := cmp.Diff(wantNames, gotNames, sortStrings); diff != "" {
			t.Fatalf("list hub nodes: unexpected response (-want +got):\n%s", diff)
		}
	})

}

func newCtx(t *testing.T) (context.Context, context.CancelFunc) {
	deadline, ok := t.Deadline()
	if !ok {
		return context.WithCancel(context.Background())
	}
	return context.WithDeadline(context.Background(), deadline)
}

func zip[T any](a, b []T) [][2]T {
	if len(a) != len(b) {
		panic("zip: slices have different lengths")
	}
	z := make([][2]T, len(a))
	for i := range a {
		z[i] = [2]T{a[i], b[i]}
	}
	return z
}
