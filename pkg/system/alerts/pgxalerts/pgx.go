package pgxalerts

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/pkg/minibus"
	"github.com/smart-core-os/sc-bos/pkg/proto/alertpb"
	"github.com/smart-core-os/sc-golang/pkg/masks"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

//go:embed schema.sql
var schemaSql string

func SetupDB(ctx context.Context, pool *pgxpool.Pool) error {
	return pool.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, schemaSql)
		return err
	})
}

func NewServer(ctx context.Context, connStr string) (*Server, error) {
	pool, err := pgxpool.Connect(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("connect %w", err)
	}

	return NewServerFromPool(ctx, pool)
}

func NewServerFromPool(ctx context.Context, pool *pgxpool.Pool) (*Server, error) {
	err := SetupDB(ctx, pool)
	if err != nil {
		return nil, fmt.Errorf("setup %w", err)
	}

	return &Server{
		pool: pool,
		bus:  &minibus.Bus[*alertpb.PullAlertsResponse_Change]{},
		Severity: []alertpb.Alert_Severity{
			alertpb.Alert_INFO,
			alertpb.Alert_WARNING,
			alertpb.Alert_SEVERE,
			alertpb.Alert_LIFE_SAFETY,
		},
	}, nil
}

type Server struct {
	alertpb.UnimplementedAlertApiServer
	alertpb.UnimplementedAlertAdminApiServer

	// Floors, if set, is used to pre-populate AlertMetadata with zero values for cases when no alerts appear on a floor.
	Floors []string
	// Zones, if set, is used to pre-populate AlertMetadata with zero values for cases when no alerts appear in a zone.
	Zones []string
	// Severity, if set, is used to pre-populate AlertMetadata with zero values for cases when no alerts have a severity.
	Severity []alertpb.Alert_Severity
	// Subsystems, if set, is used to pre-populate AlertMetadata with zero values for cases when no alerts appear in a subsystem.
	Subsystems []string

	pool *pgxpool.Pool
	bus  *minibus.Bus[*alertpb.PullAlertsResponse_Change]

	// to support alert metadata
	mdMu   sync.Mutex      // guards the following md fields
	md     *resource.Value // of *alertpb.AlertMetadata, used to track changes
	mdC    chan struct{}   // nil if needs init, blocked if init-ing, closed if done
	mdErr  error           // non-nil if mdC is done and completed with error
	mdStop func()          // closes any go routines that are listening for changes
}

func (s *Server) Close() error {
	s.mdMu.Lock()
	if s.mdStop != nil {
		s.mdStop()
	}
	s.mdMu.Unlock()

	return nil
}

func (s *Server) CreateAlert(ctx context.Context, request *alertpb.CreateAlertRequest) (*alertpb.Alert, error) {
	alert := request.Alert
	if alert.Description == "" {
		return nil, status.Error(codes.InvalidArgument, "description empty")
	}
	if alert.Severity == 0 {
		alert.Severity = alertpb.Alert_WARNING
	}

	if request.MergeSource {
		if alert.GetSource() == "" {
			return nil, status.Error(codes.InvalidArgument, "source empty")
		}
		return s.mergeSourceAlert(ctx, request.Name, alert)
	}
	return s.createNewAlert(ctx, request.Name, alert)
}

func (s *Server) createNewAlert(ctx context.Context, name string, alert *alertpb.Alert) (*alertpb.Alert, error) {
	alert, err := insertAlert(ctx, s.pool, alert)
	if err != nil {
		return nil, dbErrToStatus(err)
	}

	go s.notifyAdd(name, alert)

	return alert, nil
}

func (s *Server) mergeSourceAlert(ctx context.Context, name string, alert *alertpb.Alert) (*alertpb.Alert, error) {
	var oldAlert, newAlert *alertpb.Alert
	err := s.pool.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		args := []any{
			alert.Source,
		}
		sql := selectAlertSQL + ` WHERE source = $1`
		if alert.Federation != "" {
			sql += ` AND federation = $2`
			args = append(args, alert.Federation)
		}
		// note: we don't WHERE resolve_time is NULL because we _want_ to see if the last alert was resolved or not
		sql += ` ORDER BY create_time DESC LIMIT 1 FOR UPDATE`

		oldAlert = &alertpb.Alert{}
		row := tx.QueryRow(ctx, sql, args...)
		err := scanAlert(row, oldAlert)
		switch {
		case errors.Is(err, pgx.ErrNoRows) || oldAlert.GetResolveTime() != nil:
			oldAlert = nil
			newAlert, err = insertAlert(ctx, tx, alert)
			return err
		case err != nil:
			return err
		}

		// a consequence of the below logic is that you are incapable of clearing any of these fields
		var fields []string
		var values []any
		if alert.Description != "" { // technically description will never be empty, but check for consistency
			fields = append(fields, "description")
			values = append(values, alert.Description)
		}
		if alert.Severity != 0 {
			fields = append(fields, "severity")
			values = append(values, alert.Severity)
		}
		if alert.Floor != "" {
			fields = append(fields, "floor")
			values = append(values, alert.Floor)
		}
		if alert.Zone != "" {
			fields = append(fields, "zone")
			values = append(values, alert.Zone)
		}
		if alert.Subsystem != "" {
			fields = append(fields, "subsystem")
			values = append(values, alert.Subsystem)
		}

		if len(fields) > 0 {
			if err := updateAlert(ctx, tx, oldAlert.Id, fields, values); err != nil {
				return err
			}
			newAlert = &alertpb.Alert{Id: oldAlert.Id}
			return readAlertById(ctx, tx, oldAlert.Id, newAlert)
		} else {
			newAlert = oldAlert // no change made
		}
		return nil
	})
	if err != nil {
		return nil, dbErrToStatus(err)
	}

	if oldAlert == nil {
		go s.notifyAdd(name, newAlert)
	} else if !proto.Equal(oldAlert, newAlert) {
		go s.notifyUpdate(name, oldAlert, newAlert)
	}

	return newAlert, nil
}

func (s *Server) UpdateAlert(ctx context.Context, request *alertpb.UpdateAlertRequest) (*alertpb.Alert, error) {
	alert := request.Alert
	if alert.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id empty")
	}

	var fields []string
	var values []any
	if shouldUpdateField(request.UpdateMask, "description", alert.Description) {
		fields = append(fields, "description")
		values = append(values, alert.Description)
	}
	if shouldUpdateField(request.UpdateMask, "severity", alert.Severity) {
		fields = append(fields, "severity")
		values = append(values, alert.Severity)
	}
	if shouldUpdateField(request.UpdateMask, "floor", alert.Floor) {
		fields = append(fields, "floor")
		values = append(values, alert.Floor)
	}
	if shouldUpdateField(request.UpdateMask, "zone", alert.Zone) {
		fields = append(fields, "zone")
		values = append(values, alert.Zone)
	}
	if shouldUpdateField(request.UpdateMask, "subsystem", alert.Subsystem) {
		fields = append(fields, "subsystem")
		values = append(values, alert.Subsystem)
	}
	if shouldUpdateField(request.UpdateMask, "source", alert.Source) {
		fields = append(fields, "source")
		values = append(values, alert.Source)
	}
	if shouldUpdateField(request.UpdateMask, "federation", alert.Federation) {
		fields = append(fields, "federation")
		values = append(values, alert.Federation)
	}

	if len(fields) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no fields to update")
	}

	original := &alertpb.Alert{}
	updated := &alertpb.Alert{}

	err := s.pool.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		// record the original value which will be used for event notifications
		if err := readAlertById(ctx, tx, alert.Id, original); err != nil {
			return err
		}
		if err := updateAlert(ctx, tx, alert.Id, fields, values); err != nil {
			return err
		}
		return readAlertById(ctx, tx, alert.Id, updated)
	})

	if err != nil {
		return nil, dbErrToStatus(err)
	}

	// notify
	go s.notifyUpdate(request.Name, original, updated)

	return updated, nil
}

func (s *Server) ResolveAlert(ctx context.Context, request *alertpb.ResolveAlertRequest) (*alertpb.Alert, error) {
	alert := request.Alert

	original := &alertpb.Alert{}
	updated := &alertpb.Alert{}

	respond := func(err error) (*alertpb.Alert, error) {
		switch {
		case status.Code(err) == codes.NotFound:
			if !request.AllowMissing {
				return nil, err
			}
			return original, nil
		case errors.Is(err, pgx.ErrNoRows):
			if !request.AllowMissing {
				return nil, status.Error(codes.NotFound, "alert not found")
			}
			return original, nil
		case err != nil:
			return nil, dbErrToStatus(err)
		}

		if !proto.Equal(original, updated) {
			go s.notifyUpdate(request.Name, original, updated)
		}
		return updated, nil
	}

	now := time.Now()
	setResolveTime := func(ctx context.Context, tx pgx.Tx, id string) error {
		sql := `UPDATE alerts SET resolve_time=$1 WHERE id=$2`
		res, err := tx.Exec(ctx, sql, now, id)
		if err != nil {
			return err
		}
		rows := res.RowsAffected()
		if rows == 0 {
			return status.Error(codes.NotFound, "alert not found")
		}
		// get alert so we can return it
		return readAlertById(ctx, tx, id, updated)
	}

	switch {
	case alert.Id != "":
		return respond(s.pool.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
			// record the original value which will be used for event notifications
			if err := readAlertById(ctx, tx, alert.Id, original); err != nil {
				return err
			}
			if original.ResolveTime != nil {
				// already resolved
				updated = original
				return nil
			}
			return setResolveTime(ctx, tx, alert.Id)
		}))
	case alert.Source != "":
		return respond(s.pool.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
			args := []any{alert.Source}
			querySql := selectAlertSQL + ` WHERE source=$1`
			if alert.Federation != "" {
				querySql += ` AND federation=$2`
				args = append(args, alert.Federation)
			}
			querySql += ` ORDER BY create_time DESC LIMIT 1 FOR UPDATE`
			row := tx.QueryRow(ctx, querySql, args...)
			if err := scanAlert(row, original); err != nil {
				return err
			}
			if original.ResolveTime != nil {
				// already resolved
				updated = original
				return nil
			}
			return setResolveTime(ctx, tx, original.Id)
		}))
	}

	return nil, status.Error(codes.InvalidArgument, "id and source missing")
}

func (s *Server) DeleteAlert(ctx context.Context, request *alertpb.DeleteAlertRequest) (*alertpb.DeleteAlertResponse, error) {
	if request.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id empty")
	}
	existing := &alertpb.Alert{}
	err := s.pool.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		// We do extra work to get the old value so we can include it in bus events.
		// Without this any filtered PullAlerts call wouldn't be able to correctly include the event in responses.
		err := readAlertById(ctx, tx, request.Id, existing)
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `DELETE FROM alerts WHERE id=$1`, request.Id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			if request.AllowMissing {
				return nil
			}
			return status.Error(codes.NotFound, request.Id)
		}
		return nil
	})

	if err != nil {
		err := dbErrToStatus(err)
		if status.Code(err) == codes.NotFound && request.AllowMissing {
			return &alertpb.DeleteAlertResponse{}, nil
		}
		return nil, err
	}

	// notify
	go s.notifyRemove(request.Name, existing)

	return &alertpb.DeleteAlertResponse{}, nil
}

func (s *Server) ListAlerts(ctx context.Context, request *alertpb.ListAlertsRequest) (*alertpb.ListAlertsResponse, error) {
	var args []any
	var argIdx int
	var where []string // combined with AND
	if request.PageToken != "" {
		pt, err := DecodePageToken(request.PageToken)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "bad page token")
		}
		where = append(where, fmt.Sprintf(`create_time<$%d OR (create_time=$%d AND id<=$%d)`, argIdx+1, argIdx+1, argIdx+2))
		args = append(args, pt.LastCreateTime, pt.LastID)
		argIdx += 2
	}
	if request.Query != nil {
		q := request.Query

		if q.CreatedNotBefore != nil {
			where = append(where, fmt.Sprintf(`create_time>=$%d`, argIdx+1))
			args = append(args, q.CreatedNotBefore.AsTime())
			argIdx += 1
		}
		if q.CreatedNotAfter != nil {
			where = append(where, fmt.Sprintf(`create_time<=$%d`, argIdx+1))
			args = append(args, q.CreatedNotAfter.AsTime())
			argIdx += 1
		}
		if q.SeverityNotBelow != 0 {
			where = append(where, fmt.Sprintf(`severity>=$%d`, argIdx+1))
			args = append(args, q.SeverityNotBelow)
			argIdx += 1
		}
		if q.SeverityNotAbove != 0 {
			where = append(where, fmt.Sprintf(`severity<=$%d`, argIdx+1))
			args = append(args, q.SeverityNotAbove)
			argIdx += 1
		}
		if q.Floor != "" {
			where = append(where, fmt.Sprintf(`floor=$%d`, argIdx+1))
			args = append(args, q.Floor)
			argIdx += 1
		}
		if q.Zone != "" {
			where = append(where, fmt.Sprintf(`zone=$%d`, argIdx+1))
			args = append(args, q.Zone)
			argIdx += 1
		}
		if q.Subsystem != "" {
			where = append(where, fmt.Sprintf(`subsystem=$%d`, argIdx+1))
			args = append(args, q.Subsystem)
			argIdx += 1
		}
		if q.Source != "" {
			where = append(where, fmt.Sprintf(`source=$%d`, argIdx+1))
			args = append(args, q.Source)
			argIdx += 1
		}
		if q.Federation != "" {
			where = append(where, fmt.Sprintf(`federation=$%d`, argIdx+1))
			args = append(args, q.Federation)
			argIdx += 1
		}
		if q.Acknowledged != nil {
			if *q.Acknowledged {
				where = append(where, `ack_time IS NOT NULL`)
			} else {
				where = append(where, `ack_time IS NULL`)
			}
		}
		if q.Resolved != nil {
			if *q.Resolved {
				where = append(where, `resolve_time IS NOT NULL`)
			} else {
				where = append(where, `resolve_time IS NULL`)
			}
		}
		if q.ResolvedNotBefore != nil {
			where = append(where, fmt.Sprintf(`resolve_time>=$%d`, argIdx+1))
			args = append(args, q.ResolvedNotBefore.AsTime())
			argIdx += 1
		}
		if q.ResolvedNotAfter != nil {
			where = append(where, fmt.Sprintf(`resolve_time<=$%d`, argIdx+1))
			args = append(args, q.ResolvedNotAfter.AsTime())
			argIdx += 1
		}
	}

	sql := selectAlertSQL
	if len(where) > 0 {
		sql += " WHERE (" + strings.Join(where, ") AND (") + ")"
	}
	sql += " ORDER BY create_time DESC, id DESC"
	// +1 so we read past the end of the page to know if theres another page or not
	pageSize := normalizePageSize(request.PageSize)
	sql += fmt.Sprintf(" LIMIT %d", pageSize+1)

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, dbErrToStatus(err)
	}
	defer rows.Close()

	var alerts []*alertpb.Alert
	for rows.Next() {
		alert := &alertpb.Alert{}
		err := scanAlert(rows, alert)
		if err != nil {
			return nil, dbErrToStatus(err)
		}
		alerts = append(alerts, alert)
	}

	res := &alertpb.ListAlertsResponse{}
	if len(alerts) <= pageSize {
		// no more pages, remember we selected pageSize+1 rows so we could be sure.
		res.Alerts = alerts
		return res, nil
	}

	// else there are more pages
	// calc the next page token
	res.Alerts = alerts[:pageSize]
	lastRecord := alerts[len(alerts)-1]
	npt := PageToken{
		LastCreateTime: lastRecord.CreateTime.AsTime(),
		LastID:         lastRecord.Id,
	}
	nptStr, err := npt.Encode()
	if err != nil {
		return nil, err
	}
	res.NextPageToken = nptStr

	filter := masks.NewResponseFilter(masks.WithFieldMask(request.ReadMask))
	for _, alert := range res.Alerts {
		filter.Filter(alert)
	}
	return res, nil
}

func (s *Server) PullAlerts(request *alertpb.PullAlertsRequest, server alertpb.AlertApi_PullAlertsServer) error {
	filter := masks.NewResponseFilter(masks.WithFieldMask(request.ReadMask))
	for change := range s.bus.Listen(server.Context()) {
		change := convertChangeForQuery(request.Query, change)
		if change == nil {
			continue
		}
		if change.OldValue != nil {
			change.OldValue = filter.FilterClone(change.OldValue).(*alertpb.Alert)
		}
		if change.NewValue != nil {
			change.NewValue = filter.FilterClone(change.NewValue).(*alertpb.Alert)
		}
		err := server.Send(&alertpb.PullAlertsResponse{Changes: []*alertpb.PullAlertsResponse_Change{change}})
		if err != nil {
			return err
		}
	}
	return server.Context().Err()
}

func (s *Server) AcknowledgeAlert(ctx context.Context, request *alertpb.AcknowledgeAlertRequest) (*alertpb.Alert, error) {
	if request.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id empty")
	}

	existing := &alertpb.Alert{}
	updated := &alertpb.Alert{}
	err := s.pool.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		err := readAlertById(ctx, tx, request.Id, existing)
		if err != nil {
			return err
		}

		if existing.Acknowledgement != nil {
			if request.AllowAcknowledged {
				updated = existing
				return nil
			}
			return status.Error(codes.Aborted, "already acknowledged")
		}

		var ackAuthorId, ackAuthorName, ackAuthorEmail *string
		if request.Author != nil {
			a := request.Author
			if a.Id != "" {
				ackAuthorId = &a.Id
			}
			if a.DisplayName != "" {
				ackAuthorName = &a.DisplayName
			}
			if a.Email != "" {
				ackAuthorEmail = &a.Email
			}
		}
		_, err = tx.Exec(ctx, `UPDATE alerts SET ack_time=now(), ack_author_id=$2, ack_author_name=$3, ack_author_email=$4 WHERE id=$1`,
			request.Id, ackAuthorId, ackAuthorName, ackAuthorEmail)
		if err != nil {
			return err
		}

		return readAlertById(ctx, tx, request.Id, updated)
	})

	if err != nil {
		err := dbErrToStatus(err)
		if status.Code(err) == codes.NotFound && request.AllowMissing {
			return &alertpb.Alert{}, nil
		}
		return nil, err
	}

	if !proto.Equal(existing, updated) {
		// notify
		go s.notifyUpdate(request.Name, existing, updated)
	}

	return updated, nil
}

func (s *Server) UnacknowledgeAlert(ctx context.Context, request *alertpb.AcknowledgeAlertRequest) (*alertpb.Alert, error) {
	if request.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id empty")
	}

	existing := &alertpb.Alert{}
	updated := &alertpb.Alert{}
	err := s.pool.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		err := readAlertById(ctx, tx, request.Id, existing)
		if err != nil {
			return err
		}

		if existing.Acknowledgement == nil {
			if request.AllowAcknowledged {
				updated = existing
				return nil
			}
			return status.Error(codes.Aborted, "already not acknowledged")
		}

		_, err = tx.Exec(ctx, `UPDATE alerts SET ack_time=null, ack_author_id=null, ack_author_name=null, ack_author_email=null WHERE id=$1`, request.Id)
		if err != nil {
			return err
		}

		return readAlertById(ctx, tx, request.Id, updated)
	})

	if err != nil {
		err := dbErrToStatus(err)
		if status.Code(err) == codes.NotFound && request.AllowMissing {
			return &alertpb.Alert{}, nil
		}
		return nil, err
	}

	if !proto.Equal(existing, updated) {
		// notify
		go s.notifyUpdate(request.Name, existing, updated)
	}

	return updated, nil
}

func dbErrToStatus(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return status.Error(codes.NotFound, "alert")
	}
	if pgerr, ok := err.(*pgconn.PgError); ok {
		if strings.HasPrefix(pgerr.Code, "22") { // data error
			// If we're getting data errors involving uuids then someone is giving us an id string that doesn't match
			// the uuid format, we don't care that it doesn't match, but we should return NotFound in this case.
			if strings.Contains(pgerr.Message, "uuid") {
				return status.Error(codes.NotFound, "id")
			}
			return status.Error(codes.InvalidArgument, pgerr.Error())
		}
	}
	// todo: convert err to a more meaningful status code
	return err
}

// shouldUpdateField returns true if the given field value should be written to the db given the field mask.
// If m is nil then only non-zero values should be written, otherwise fieldMaskIncludesPath is consulted.
func shouldUpdateField[T comparable](m *fieldmaskpb.FieldMask, f string, v T) bool {
	if m == nil {
		var zero T
		return v != zero
	}
	return fieldMaskIncludesPath(m, f)
}

// fieldMaskIncludesPath returns true if the given FieldMask indicates that p should be included.
// This happens if m is nil, m.Paths includes p, or p is a more specific path for one in m.
func fieldMaskIncludesPath(m *fieldmaskpb.FieldMask, p string) bool {
	if m == nil {
		return true
	}
	i := fieldmaskpb.Intersect(m, &fieldmaskpb.FieldMask{Paths: []string{p}})
	return len(i.Paths) > 0
}

func convertChangeForQuery(q *alertpb.Alert_Query, change *alertpb.PullAlertsResponse_Change) *alertpb.PullAlertsResponse_Change {
	if q == nil {
		return change
	}

	res := proto.Clone(change).(*alertpb.PullAlertsResponse_Change)
	if change.OldValue != nil && !alertMatchesQuery(q, change.OldValue) {
		res.OldValue = nil
	}
	if change.NewValue != nil && !alertMatchesQuery(q, change.NewValue) {
		res.NewValue = nil
	}
	if res.OldValue == nil && res.NewValue == nil {
		return nil // change should be ignored
	}
	if res.NewValue == nil {
		// delete
		res.Type = types.ChangeType_REMOVE
		return res
	}
	if res.OldValue == nil {
		// create
		res.Type = types.ChangeType_ADD
		return res
	}
	// else update
	res.Type = types.ChangeType_UPDATE
	return res
}

func alertMatchesQuery(q *alertpb.Alert_Query, a *alertpb.Alert) bool {
	if q == nil {
		return true
	}
	if a == nil {
		return false
	}

	if q.CreatedNotBefore != nil {
		if a.CreateTime == nil {
			return false
		}
		if a.CreateTime.AsTime().Before(q.CreatedNotBefore.AsTime()) {
			return false
		}
	}
	if q.CreatedNotAfter != nil {
		if a.CreateTime == nil {
			return false
		}
		if a.CreateTime.AsTime().After(q.CreatedNotAfter.AsTime()) {
			return false
		}
	}
	if q.SeverityNotBelow != 0 && int32(a.Severity) < q.SeverityNotBelow {
		return false
	}
	if q.SeverityNotAbove != 0 && int32(a.Severity) > q.SeverityNotAbove {
		return false
	}
	if q.Floor != "" && q.Floor != a.Floor {
		return false
	}
	if q.Zone != "" && q.Zone != a.Zone {
		return false
	}
	if q.Subsystem != "" && q.Subsystem != a.Subsystem {
		return false
	}
	if q.Source != "" && q.Source != a.Source {
		return false
	}
	if q.Federation != "" && q.Federation != a.Federation {
		return false
	}

	if q.Acknowledged != nil {
		wantAck := *q.Acknowledged
		if wantAck && a.Acknowledgement == nil {
			return false
		}
		if !wantAck && a.Acknowledgement != nil {
			return false
		}
	}
	if q.Resolved != nil {
		wantResolved := *q.Resolved
		if wantResolved && a.ResolveTime == nil {
			return false
		}
		if !wantResolved && a.ResolveTime != nil {
			return false
		}
	}
	if q.ResolvedNotBefore != nil {
		if a.ResolveTime == nil {
			return false
		}
		if a.ResolveTime.AsTime().Before(q.ResolvedNotBefore.AsTime()) {
			return false
		}
	}
	if q.ResolvedNotAfter != nil {
		if a.ResolveTime == nil {
			return false
		}
		if a.ResolveTime.AsTime().After(q.ResolvedNotAfter.AsTime()) {
			return false
		}
	}

	return true
}
