package hubalerts

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/alertpb"
	"github.com/smart-core-os/sc-bos/pkg/system/alerts/alertmd"
	"github.com/smart-core-os/sc-bos/pkg/util/once"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// Server backs off to a remove server, modifying requests to filter by federation parameter.
type Server struct {
	alertpb.UnimplementedAlertApiServer
	alertpb.UnimplementedAlertAdminApiServer

	remoteName string // can be empty
	federation string
	remoteNode node.Remote

	alertsOnce once.RetryError
	alerts     alertpb.AlertApiClient
	alertAdmin alertpb.AlertAdminApiClient

	// to support alert metadata, which we have to track ourselves
	mdOnce once.RetryError
	md     *resource.Value // of *alertpb.AlertMetadata, used to track changes
	mdStop func()          // closes any go routines that are listening for changes
}

func NewServer(remoteName, localName string, remoteNode node.Remote) *Server {
	return &Server{
		remoteName: remoteName,
		federation: localName,
		remoteNode: remoteNode,
	}
}

func (s *Server) CreateAlert(ctx context.Context, request *alertpb.CreateAlertRequest) (*alertpb.Alert, error) {
	if err := s.initConn(ctx); err != nil {
		return nil, err
	}
	request.Name = s.remoteName
	request.Alert.Federation = s.federation
	return s.alertAdmin.CreateAlert(ctx, request)
}

func (s *Server) UpdateAlert(ctx context.Context, request *alertpb.UpdateAlertRequest) (*alertpb.Alert, error) {
	if err := s.initConn(ctx); err != nil {
		return nil, err
	}
	request.Name = s.remoteName
	request.Alert.Federation = s.federation
	return s.alertAdmin.UpdateAlert(ctx, request)
}

func (s *Server) ResolveAlert(ctx context.Context, request *alertpb.ResolveAlertRequest) (*alertpb.Alert, error) {
	if err := s.initConn(ctx); err != nil {
		return nil, err
	}
	request.Name = s.remoteName
	request.Alert.Federation = s.federation
	return s.alertAdmin.ResolveAlert(ctx, request)
}

func (s *Server) DeleteAlert(ctx context.Context, request *alertpb.DeleteAlertRequest) (*alertpb.DeleteAlertResponse, error) {
	if err := s.initConn(ctx); err != nil {
		return nil, err
	}
	request.Name = s.remoteName
	return s.alertAdmin.DeleteAlert(ctx, request)
}

func (s *Server) ListAlerts(ctx context.Context, request *alertpb.ListAlertsRequest) (*alertpb.ListAlertsResponse, error) {
	if err := s.initConn(ctx); err != nil {
		return nil, err
	}
	request.Name = s.remoteName
	if request.Query == nil {
		request.Query = &alertpb.Alert_Query{}
	}
	request.Query.Federation = s.federation
	return s.alerts.ListAlerts(ctx, request)
}

func (s *Server) PullAlerts(request *alertpb.PullAlertsRequest, server alertpb.AlertApi_PullAlertsServer) error {
	if err := s.initConn(server.Context()); err != nil {
		return err
	}
	request.Name = s.remoteName
	if request.Query == nil {
		request.Query = &alertpb.Alert_Query{}
	}
	request.Query.Federation = s.federation
	stream, err := s.alerts.PullAlerts(server.Context(), request)
	if err != nil {
		return err
	}
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		err = server.Send(msg)
		if err != nil {
			return err
		}
	}
}

func (s *Server) AcknowledgeAlert(ctx context.Context, request *alertpb.AcknowledgeAlertRequest) (*alertpb.Alert, error) {
	if err := s.initConn(ctx); err != nil {
		return nil, err
	}
	request.Name = s.remoteName
	return s.alerts.AcknowledgeAlert(ctx, request)
}

func (s *Server) UnacknowledgeAlert(ctx context.Context, request *alertpb.AcknowledgeAlertRequest) (*alertpb.Alert, error) {
	if err := s.initConn(ctx); err != nil {
		return nil, err
	}
	request.Name = s.remoteName
	return s.alerts.UnacknowledgeAlert(ctx, request)
}

func (s *Server) GetAlertMetadata(ctx context.Context, request *alertpb.GetAlertMetadataRequest) (*alertpb.AlertMetadata, error) {
	if err := s.initAlertMetadata(ctx); err != nil {
		return nil, err
	}
	return s.md.Get(resource.WithReadMask(request.ReadMask)).(*alertpb.AlertMetadata), nil
}

func (s *Server) PullAlertMetadata(request *alertpb.PullAlertMetadataRequest, server alertpb.AlertApi_PullAlertMetadataServer) error {
	if err := s.initAlertMetadata(server.Context()); err != nil {
		return err
	}
	for change := range s.md.Pull(server.Context(), resource.WithReadMask(request.GetReadMask()), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&alertpb.PullAlertMetadataResponse{Changes: []*alertpb.PullAlertMetadataResponse_Change{
			{
				Name:       request.Name,
				ChangeTime: timestamppb.New(change.ChangeTime),
				Metadata:   change.Value.(*alertpb.AlertMetadata),
			},
		}})
		if err != nil {
			return err
		}
	}

	return nil
}

// initConn sets up the remote clients alerts and alertAdmin.
func (s *Server) initConn(ctx context.Context) error {
	return s.alertsOnce.Do(ctx, func() error {
		conn, err := s.remoteNode.Connect(ctx)
		if err != nil {
			return err
		}
		s.alerts = alertpb.NewAlertApiClient(conn)
		s.alertAdmin = alertpb.NewAlertAdminApiClient(conn)
		return nil
	})
}

// initAlertMetadata seeds s.md and keeps it up to date with changes to the underlying store
// if s.md hasn't already been seeded.
// initAlertMetadata blocks until either s.md is seeded or an error occurs or ctx expires.
// If an error occurs seeding s.md then it is returned
func (s *Server) initAlertMetadata(ctx context.Context) error {
	if err := s.initConn(ctx); err != nil {
		return err
	}
	return s.mdOnce.Do(ctx, func() error {
		var ctx context.Context
		ctx, s.mdStop = context.WithCancel(context.Background())

		// Collect initial stats from the DB
		md := alertmd.New()
		val := resource.NewValue(resource.WithInitialValue(md))

		// We need to pull and list the alerts from the server.
		// Normally we'd be able to do this via a single pull call but the pgx server doesn't support that :(
		// For now we'll have to live with this race condition
		query := &alertpb.Alert_Query{Federation: s.federation}
		stream, err := s.alerts.PullAlerts(ctx, &alertpb.PullAlertsRequest{Name: s.remoteName, Query: query})
		if err != nil {
			return err
		}
		listReq := &alertpb.ListAlertsRequest{Name: s.remoteName, Query: query}
		for {
			list, err := s.alerts.ListAlerts(ctx, listReq)
			if err != nil {
				return err
			}
			listReq.PageToken = list.NextPageToken

			for _, alert := range list.Alerts {
				err := alertmd.ApplyMdDelta(val, &alertpb.PullAlertsResponse_Change{
					Type:     types.ChangeType_ADD,
					NewValue: alert,
				})
				if err != nil {
					return err
				}
			}

			if listReq.PageToken == "" {
				break
			}
		}

		s.md = val

		// setup listeners for changes so we can track those changes in metadata
		go func() {
			for {
				msg, err := stream.Recv()
				if err != nil {
					stream, err = s.pullAlertsAgain(ctx, &alertpb.PullAlertsRequest{Name: s.remoteName, Query: query})
					if err != nil {
						return // ctx done, aka server stopped
					}
					continue // receive from the new stream
				}

				for _, change := range msg.Changes {
					alertmd.ApplyMdDelta(s.md, change)
				}
			}
		}()
		return nil
	})
}

// pullAlertsAgain calls s.alerts.PullAlerts, retrying on failure until ctx is done.
func (s *Server) pullAlertsAgain(ctx context.Context, req *alertpb.PullAlertsRequest) (alertpb.AlertApi_PullAlertsClient, error) {
	scale := 1.1
	max := 20 * time.Second
	delay := 10 * time.Millisecond
	for {
		select {
		case <-ctx.Done():
			// this is the only non-successful return we have
			return nil, ctx.Err()
		default:
		}

		stream, err := s.alerts.PullAlerts(ctx, req)
		if err == nil {
			return stream, nil
		}

		time.Sleep(delay)
		delay = time.Duration(float64(delay) * scale)
		if delay > max {
			delay = max
		}
	}
}
