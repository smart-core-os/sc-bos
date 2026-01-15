package pgxalerts

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/alertpb"
)

// selectAlertSQL selects fields in the order expected by scanAlert.
const selectAlertSQL = `SELECT id, description, severity, create_time, resolve_time, floor, zone, subsystem, source, federation, ack_time, ack_author_id, ack_author_name, ack_author_email FROM alerts`

type QueryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func insertAlert(ctx context.Context, q QueryRower, alert *alertpb.Alert) (*alertpb.Alert, error) {
	var createTime time.Time
	err := q.QueryRow(ctx,
		`INSERT INTO alerts (description, severity, floor, zone, subsystem, source, federation) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, create_time`,
		alert.Description, alert.Severity, alert.Floor, alert.Zone, alert.Subsystem, alert.Source, alert.Federation,
	).Scan(&alert.Id, &createTime)
	if err != nil {
		return nil, err
	}

	alert.CreateTime = timestamppb.New(createTime)
	return alert, nil
}

type Execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func updateAlert(ctx context.Context, ex Execer, id string, fields []string, values []any) error {
	setStr := ""
	for i, field := range fields {
		if setStr != "" {
			setStr += ","
		}
		setStr += fmt.Sprintf("%s=$%d", field, i+2) // +2 to make it 1-based leaving room for $1 to be the id
	}
	args := append([]any{id}, values...)
	sql := fmt.Sprintf(`UPDATE alerts SET %s WHERE id=$1`, setStr)
	res, err := ex.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	rows := res.RowsAffected()
	if rows == 0 {
		return status.Error(codes.NotFound, "alert not found")
	}
	return nil
}

func readAlertById(ctx context.Context, tx QueryRower, id string, dst *alertpb.Alert) error {
	row := tx.QueryRow(ctx,
		selectAlertSQL+` WHERE id=$1`,
		id,
	)
	return scanAlert(row, dst)
}

func scanAlert(scanner pgx.Row, dst *alertpb.Alert) error {
	var createTime, resolveTime, ackTime *time.Time
	var floor, zone, subsystem, source, federation *string
	var ackAuthorId, ackAuthorName, ackAuthorEmail *string
	err := scanner.Scan(&dst.Id, &dst.Description, &dst.Severity, &createTime, &resolveTime, &floor, &zone, &subsystem, &source, &federation, &ackTime, &ackAuthorId, &ackAuthorName, &ackAuthorEmail)
	if err != nil {
		return err
	}
	if floor != nil {
		dst.Floor = *floor
	}
	if zone != nil {
		dst.Zone = *zone
	}
	if subsystem != nil {
		dst.Subsystem = *subsystem
	}
	if source != nil {
		dst.Source = *source
	}
	if federation != nil {
		dst.Federation = *federation
	}
	if createTime != nil {
		dst.CreateTime = timestamppb.New(*createTime)
	}
	if resolveTime != nil {
		dst.ResolveTime = timestamppb.New(*resolveTime)
	}
	if ackTime != nil {
		dst.Acknowledgement = &alertpb.Alert_Acknowledgement{
			AcknowledgeTime: timestamppb.New(*ackTime),
		}

		// ack author details, we assume there is an author only if the author has any information
		var hasAuthor bool
		author := &alertpb.Alert_Acknowledgement_Author{}
		if ackAuthorId != nil {
			hasAuthor = true
			author.Id = *ackAuthorId
		}
		if ackAuthorName != nil {
			hasAuthor = true
			author.DisplayName = *ackAuthorName
		}
		if ackAuthorEmail != nil {
			hasAuthor = true
			author.Email = *ackAuthorEmail
		}
		if hasAuthor {
			dst.Acknowledgement.Author = author
		}
	}
	return nil
}
