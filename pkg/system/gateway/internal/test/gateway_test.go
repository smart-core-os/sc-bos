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

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/internal/util/grpc/reflectionapi"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/system/gateway/internal/test/shared"
	"github.com/smart-core-os/sc-golang/pkg/trait"
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
		addr := addr
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
		t.Fatal("run failed", err)
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

	client := traits.NewMetadataApiClient(conn)
	err = backoff.Retry(func() error {
		_, err := client.GetMetadata(ctx, &traits.GetMetadataRequest{})
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

	client := gen.NewHubApiClient(hubConn)
	for _, addr := range shared.ACGRPCAddrs {
		_, err := client.EnrollHubNode(ctx, &gen.EnrollHubNodeRequest{Node: &gen.HubNode{Address: addr}})
		checkErr(addr, err)
	}
	for _, addr := range shared.GWGRPCAddrs {
		_, err := client.EnrollHubNode(ctx, &gen.EnrollHubNodeRequest{Node: &gen.HubNode{Address: addr}})
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
		client := gen.NewDevicesApiClient(conn)
		testDevicesApiHasNames(t, ctx, addr, onOffDevices, client, &gen.ListDevicesRequest{
			Query: &gen.Device_Query{Conditions: []*gen.Device_Query_Condition{
				{Field: "metadata.traits.name", Value: &gen.Device_Query_Condition_StringEqual{StringEqual: string(trait.OnOff)}},
			}},
		})
	})

	t.Run("onOff devices respond", func(t *testing.T) {
		client := traits.NewOnOffApiClient(conn)
		for _, name := range onOffDevices {
			testOnOffApi(t, ctx, addr, name, client)
		}
	})
	t.Run("services respond", func(t *testing.T) {
		client := gen.NewServicesApiClient(conn)
		for _, name := range serviceDevices {
			testServicesApi(t, ctx, addr, name, client)
		}
	})
	t.Run("has health check", func(t *testing.T) {
		client := gen.NewHealthApiClient(conn)
		for _, name := range onOffDevices {
			testHealthApi(t, ctx, addr, name, client)
		}
	})
	t.Run("devices api has health checks", func(t *testing.T) {
		client := gen.NewDevicesApiClient(conn)
		testDevicesApiHasNames(t, ctx, addr, onOffDevices, client, &gen.ListDevicesRequest{
			Query: &gen.Device_Query{Conditions: []*gen.Device_Query_Condition{
				{Field: "health_checks.bounds.normal_value.string_value", Value: &gen.Device_Query_Condition_StringEqual{StringEqual: "ON"}},
			}},
		})
	})

	t.Run("reflection", func(t *testing.T) {
		testReflection(t, ctx, conn)
	})

	t.Run("stable device list", func(t *testing.T) {
		testStableDeviceList(t, ctx, conn)
	})

	testHubApis(t, ctx, conn)
}

func waitForDevice(t *testing.T, ctx context.Context, conn *grpc.ClientConn, name string) {
	t.Helper()

	client := traits.NewMetadataApiClient(conn)
	err := backoff.Retry(func() error {
		_, err := client.GetMetadata(ctx, &traits.GetMetadataRequest{Name: name})
		return err
	}, backoff.WithContext(backoff.NewExponentialBackOff(backoff.WithMaxInterval(5*time.Second)), ctx))
	if err != nil {
		t.Fatalf("wait for device %s: %v", name, err)
	}
}

func testDevicesApiHasNames(t *testing.T, ctx context.Context, addr string, names []string, client gen.DevicesApiClient, request *gen.ListDevicesRequest) {
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

func testOnOffApi(t *testing.T, ctx context.Context, addr, name string, client traits.OnOffApiClient) {
	t.Helper()

	// useful for cancelling the stream
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// set initial known state: ON
	res, err := client.UpdateOnOff(ctx, &traits.UpdateOnOffRequest{Name: name, OnOff: &traits.OnOff{State: traits.OnOff_ON}})
	if err != nil {
		t.Fatalf("[%s] update onoff %s: %v", addr, name, err)
	}
	if diff := cmp.Diff(&traits.OnOff{State: traits.OnOff_ON}, res, protocmp.Transform()); diff != "" {
		t.Fatalf("[%s] update onoff %s: unexpected response (-want +got):\n%s", addr, name, diff)
	}

	// subscribe
	changes := make(chan *traits.PullOnOffResponse, 1) // we're only expecting 1
	stream, err := client.PullOnOff(ctx, &traits.PullOnOffRequest{Name: name, UpdatesOnly: true})
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
	res, err = client.GetOnOff(ctx, &traits.GetOnOffRequest{Name: name})
	if err != nil {
		t.Fatalf("[%s] get onoff %s: %v", addr, name, err)
	}
	if diff := cmp.Diff(&traits.OnOff{State: traits.OnOff_ON}, res, protocmp.Transform()); diff != "" {
		t.Fatalf("[%s] get onoff %s: unexpected response (-want +got):\n%s", addr, name, diff)
	}

	// Perform update and check for change
	res, err = client.UpdateOnOff(ctx, &traits.UpdateOnOffRequest{Name: name, OnOff: &traits.OnOff{State: traits.OnOff_OFF}})
	if err != nil {
		t.Fatalf("[%s] update onoff %s: %v", addr, name, err)
	}
	if diff := cmp.Diff(&traits.OnOff{State: traits.OnOff_OFF}, res, protocmp.Transform()); diff != "" {
		t.Fatalf("[%s] update onoff %s: unexpected response (-want +got):\n%s", addr, name, diff)
	}
	select {
	case res := <-changes:
		want := &traits.PullOnOffResponse{Changes: []*traits.PullOnOffResponse_Change{
			{
				Name:  name,
				OnOff: &traits.OnOff{State: traits.OnOff_OFF},
			},
		}}
		// clear timestamps to make comparing easier
		for i := range res.Changes {
			res.Changes[i].ChangeTime = nil
		}
		if diff := cmp.Diff(want, res, protocmp.Transform()); diff != "" {
			t.Fatalf("[%s] pull onoff %s: unexpected response (-want +got):\n%s", addr, name, diff)
		}
	}
}

func testServicesApi(t *testing.T, ctx context.Context, addr, name string, client gen.ServicesApiClient) {
	t.Helper()

	_, err := client.ListServices(ctx, &gen.ListServicesRequest{Name: name})
	if err != nil {
		t.Fatalf("[%s] list services %s: %v", addr, name, err)
	}
}

func testHealthApi(t *testing.T, ctx context.Context, addr, name string, client gen.HealthApiClient) {
	t.Helper()
	res, err := client.ListHealthChecks(ctx, &gen.ListHealthChecksRequest{Name: name})
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
			break
		}
	}
	if !foundOnOffCheck {
		t.Fatalf("[%s] list health checks %s: no OnOff=ON health check found", addr, name)
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
		{Name: "smartcore.bos.DevicesApi"},
		{Name: "smartcore.bos.EnrollmentApi"},
		{Name: "smartcore.bos.HealthApi"},
		{Name: "smartcore.bos.HealthHistory"},
		{Name: "smartcore.bos.HubApi"},
		{Name: "smartcore.bos.ServicesApi"},
		{Name: "smartcore.traits.MetadataApi"},
		{Name: "smartcore.traits.OnOffApi"},
		{Name: "smartcore.traits.OnOffInfo"},
		{Name: "smartcore.traits.ParentApi"},
	}
	if diff := cmp.Diff(wantServices, services, protocmp.Transform()); diff != "" {
		t.Fatalf("services: (-want +got):\n%s", diff)
	}

	types := []string{
		"smartcore.traits.OnOffApi",
		"smartcore.bos.DevicesApi",
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
	client := gen.NewDevicesApiClient(conn)
	type totals struct{ add, update, remove, replace int }
	// used to track what is being unstable,
	// technically we could fail on the first event,
	// but this way gives more info about what is unstable.
	events := make(map[string]totals)
	// this stream shouldn't receive anything
	openStream := func() grpc.ServerStreamingClient[gen.PullDevicesResponse] {
		stream, err := client.PullDevices(ctx, &gen.PullDevicesRequest{UpdatesOnly: true})
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
			case types.ChangeType_ADD:
				total.add++
			case types.ChangeType_UPDATE:
				total.update++
			case types.ChangeType_REMOVE:
				total.remove++
			case types.ChangeType_REPLACE:
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
		client := gen.NewHubApiClient(conn)
		res, err := client.ListHubNodes(ctx, &gen.ListHubNodesRequest{})
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
