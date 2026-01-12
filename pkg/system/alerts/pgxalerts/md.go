package pgxalerts

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/alertpb"
	"github.com/smart-core-os/sc-bos/pkg/system/alerts/alertmd"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

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

// initAlertMetadata seeds s.md and keeps it up to date with changes to the underlying store
// if s.md hasn't already been seeded.
// initAlertMetadata blocks until either s.md is seeded or an error occurs or ctx expires.
// If an error occurs seeding s.md then it is returned
func (s *Server) initAlertMetadata(ctx context.Context) error {
	s.mdMu.Lock()
	defer s.mdMu.Unlock()
	wait := func() error {
		select {
		case <-s.mdC:
			return s.mdErr
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if s.mdC != nil {
		// init is either in progress or has completed
		return wait()
	}

	// need to init the alert md
	s.mdC = make(chan struct{})
	ctx, s.mdStop = context.WithCancel(context.Background())
	go func() {
		// no need to lock here as we already know we're the only ones writing to mdC or mdErr
		defer close(s.mdC)

		// These are populated in the db transaction to make sure we don't miss any events.
		// There is a race condition here, though very unlikely (I think).
		// There is space between our transaction and other RPC transactions (like AckAlert)
		// for events to be missed or duplicated.
		// I apologise to any future maintainer who find this comment and has to fix it, likely under pressure.
		var events <-chan *alertpb.PullAlertsResponse_Change
		var eventsCtx context.Context
		var eventsCancel context.CancelFunc

		// Collect initial stats from the DB
		md := alertmd.New()
		err := s.pool.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
			// totals
			err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM alerts`).Scan(&md.TotalCount)
			if err != nil {
				return fmt.Errorf("count all %w", err)
			}

			// ack and nak counts
			var ackCount uint32
			err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM alerts WHERE ack_time IS NOT NULL`).Scan(&ackCount)
			if err != nil {
				return fmt.Errorf("nak count %w", err)
			}
			md.AcknowledgedCounts = map[bool]uint32{
				true:  ackCount,
				false: md.TotalCount - ackCount,
			}
			// resolved and unresolved counts
			var resolveCounts uint32
			err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM alerts WHERE resolve_time IS NOT NULL`).Scan(&resolveCounts)
			if err != nil {
				return fmt.Errorf("resolve count %w", err)
			}
			md.ResolvedCounts = map[bool]uint32{
				true:  resolveCounts,
				false: md.TotalCount - resolveCounts,
			}

			// needs attention counts
			var needsAttentionCount uint32
			err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM alerts WHERE ack_time IS NOT NULL AND resolve_time IS NOT NULL`).Scan(&needsAttentionCount)
			if err != nil {
				return fmt.Errorf("ack and resolved count %w", err)
			}
			md.NeedsAttentionCounts["ack_resolved"] = needsAttentionCount
			err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM alerts WHERE ack_time IS NOT NULL AND resolve_time IS NULL`).Scan(&needsAttentionCount)
			if err != nil {
				return fmt.Errorf("ack and unresolved count %w", err)
			}
			md.NeedsAttentionCounts["ack_unresolved"] = needsAttentionCount
			err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM alerts WHERE ack_time IS NULL AND resolve_time IS NOT NULL`).Scan(&needsAttentionCount)
			if err != nil {
				return fmt.Errorf("ack and resolved count %w", err)
			}
			md.NeedsAttentionCounts["nack_resolved"] = needsAttentionCount
			err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM alerts WHERE ack_time IS NULL AND resolve_time IS NULL`).Scan(&needsAttentionCount)
			if err != nil {
				return fmt.Errorf("ack and unresolved count %w", err)
			}
			md.NeedsAttentionCounts["nack_unresolved"] = needsAttentionCount

			// floors
			floors, err := queryGroupCounts(ctx, tx, "floor", s.Floors)
			if err != nil {
				return fmt.Errorf("floors %w", err)
			}
			md.FloorCounts = floors
			// zones
			zones, err := queryGroupCounts(ctx, tx, "zone", s.Floors)
			if err != nil {
				return fmt.Errorf("zones %w", err)
			}
			md.ZoneCounts = zones
			// severity
			severity, err := queryGroupCounts(ctx, tx, "severity", s.Severity)
			if err != nil {
				return fmt.Errorf("severity %w", err)
			}
			md.SeverityCounts = severityCountMapToProto(severity)
			// subsystems
			subsystems, err := queryGroupCounts(ctx, tx, "subsystem", s.Subsystems)
			if err != nil {
				return fmt.Errorf("subsystems %w", err)
			}
			md.SubsystemCounts = subsystems

			if eventsCancel != nil {
				eventsCancel()
			}
			eventsCtx, eventsCancel = context.WithCancel(ctx)
			events = s.bus.Listen(eventsCtx)

			return nil
		})
		if err != nil {
			s.mdErr = err
			s.mdC = nil // allow another call to attempt again
			return
		}
		s.md = resource.NewValue(resource.WithInitialValue(md))

		// setup listeners for changes to the DB so we can track those changes in metadata
		go func() {
			defer eventsCancel()
			for {
				select {
				case <-ctx.Done():
					return // the server is closing
				case e := <-events:
					alertmd.ApplyMdDelta(s.md, e)
				}
			}
		}()
	}()
	return wait()
}

func queryGroupCounts[K comparable](ctx context.Context, tx pgx.Tx, col string, seed []K) (map[K]uint32, error) {
	all := make(map[K]uint32)
	for _, floor := range seed {
		all[floor] = 0
	}
	query, err := tx.Query(ctx, fmt.Sprintf(`SELECT %s, COUNT(id) FROM alerts GROUP BY %s`, col, col))
	if err != nil {
		return nil, fmt.Errorf("query %w", err)
	}
	for query.Next() {
		var id *K
		var count uint32
		err = query.Scan(&id, &count)
		if err != nil {
			return nil, fmt.Errorf("row scan %w", err)
		}
		if id == nil {
			var zero K
			id = &zero
		}
		all[*id] = count
	}
	return all, nil
}

func severityCountMapToProto(m map[alertpb.Alert_Severity]uint32) map[int32]uint32 {
	dst := make(map[int32]uint32, len(m))
	for k, v := range m {
		dst[int32(k)] = v
	}
	return dst
}
