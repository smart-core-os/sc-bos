package pgxpublications

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/publicationpb"
)

func CreatePublication(ctx context.Context, tx pgx.Tx, id string, audience string) error {
	if id == "" {
		return errors.New("publication ID cannot be empty")
	}

	// language=postgresql
	query := `
		INSERT INTO publication (id, audience_name) VALUES ($1, $2);
	`

	var nullAudience *string
	if audience != "" {
		nullAudience = &audience
	}

	_, err := tx.Exec(ctx, query, id, nullAudience)
	return err
}

func DeletePublication(ctx context.Context, tx pgx.Tx, id string, idempotent bool) error {
	// language=postgresql
	query := `
		DELETE FROM publication
		WHERE id = $1;
	`

	tag, err := tx.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 && !idempotent {
		return status.Error(codes.NotFound, "publication not found")
	}

	return nil
}

type PublicationVersion struct {
	ID            string
	PublicationID string
	PublishTime   time.Time
	Body          []byte
	MediaType     string
	Changelog     string
}

func CreatePublicationVersion(ctx context.Context, tx pgx.Tx, data PublicationVersion) (id string, err error) {
	if data.ID != "" {
		return "", errors.New("cannot add a publication version with a populated ID")
	}

	// language=postgresql
	query := `
		INSERT INTO publication_version (id, publication_id, publish_time, body, media_type, changelog)
		VALUES (DEFAULT, $1, $2, $3, $4, $5)
		RETURNING id;
	`

	var mediaType *string
	if data.MediaType != "" {
		mediaType = &data.MediaType
	}
	var changelog *string
	if data.Changelog != "" {
		changelog = &data.Changelog
	}

	row := tx.QueryRow(ctx, query,
		data.PublicationID, data.PublishTime, data.Body, mediaType, changelog)

	err = row.Scan(&id)
	return
}

func GetPublication(ctx context.Context, tx pgx.Tx, pubID, versionID string) (*publicationpb.Publication, error) {
	var row pgx.Row
	if versionID != "" {
		// fetch the version with a specific identifier
		// language=postgresql
		query := `
			SELECT p.audience_name, pv.id, pv.body, pv.publish_time, pv.media_type, a.accepted, a.receipt_time, a.rejected_reason
			FROM publication p
			INNER JOIN publication_version pv ON p.id = pv.publication_id
			LEFT JOIN publication_ack a on pv.id = a.version_id
			WHERE p.id = $1 AND pv.id = $2;
		`

		row = tx.QueryRow(ctx, query, pubID, versionID)
	} else {
		// fetch the latest version
		// language=postgresql
		query := `
			SELECT p.audience_name, pv.id, pv.body, pv.publish_time, pv.media_type, a.accepted, a.receipt_time, a.rejected_reason
			FROM publication p
			INNER JOIN publication_version pv ON p.id = pv.publication_id
			LEFT JOIN publication_ack a on pv.id = a.version_id
			WHERE p.id = $1
			ORDER BY pv.publish_time DESC
			LIMIT 1;
		`

		row = tx.QueryRow(ctx, query, pubID)
	}

	rowData := publicationVersionRow{PublicationID: pubID}
	err := row.Scan(
		&rowData.AudienceName,
		&rowData.VersionID,
		&rowData.Body,
		&rowData.PublishTime,
		&rowData.MediaType,
		&rowData.Accepted,
		&rowData.ReceiptTime,
		&rowData.RejectedReason,
	)

	if err != nil {
		return nil, err
	}

	return publicationVersionRowToProto(rowData), nil
}

func GetLatestVersionID(ctx context.Context, tx pgx.Tx, pubID string) (version string, err error) {
	// language=postgresql
	query := `
		SELECT id
		FROM publication_version
		WHERE publication_id = $1
		ORDER BY publish_time DESC
		LIMIT 1;
	`

	row := tx.QueryRow(ctx, query, pubID)
	err = row.Scan(&version)
	return
}

func GetPublicationsPaginated(ctx context.Context, tx pgx.Tx, token string, limit int) (publications []*publicationpb.Publication, nextToken string, err error) {
	// language=postgresql
	query := `
		SELECT DISTINCT ON (p.id) p.id, p.audience_name, pv.id, pv.body, pv.publish_time, pv.media_type, a.accepted, 
		                          a.receipt_time, a.rejected_reason
		FROM publication p
		INNER JOIN publication_version pv ON p.id = pv.publication_id
		LEFT JOIN publication_ack a on pv.id = a.version_id
		WHERE p.id > $1 OR $1 = ''
		ORDER BY p.id, pv.publish_time DESC
		LIMIT $2;
	`

	rows, err := tx.Query(ctx, query, token, limit)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	for rows.Next() {
		var rowData publicationVersionRow
		err = rows.Scan(
			&rowData.PublicationID,
			&rowData.AudienceName,
			&rowData.VersionID,
			&rowData.Body,
			&rowData.PublishTime,
			&rowData.MediaType,
			&rowData.Accepted,
			&rowData.ReceiptTime,
			&rowData.RejectedReason,
		)
		if err != nil {
			return nil, "", err
		}

		publications = append(publications, publicationVersionRowToProto(rowData))
		// the continuation token is simply the last publication ID to be encountered
		nextToken = rowData.PublicationID
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, "", err
	}

	return
}

func GetPublicationIDForVersion(ctx context.Context, tx pgx.Tx, versionID string) (pubID string, err error) {
	// language=postgresql
	query := "SELECT publication_id FROM publication_version WHERE id = $1;"

	row := tx.QueryRow(ctx, query, versionID)
	err = row.Scan(&pubID)
	return
}

type Acknowledgement struct {
	ID             string
	Accepted       bool
	RejectedReason string
	Time           time.Time
}

// GetAcknowledgement retrieves the acknowledgement state of a particular publication version.
// If the version has not been acknowledged, returns a nil Acknowledgement.
func GetAcknowledgement(ctx context.Context, tx pgx.Tx, versionID string) (*Acknowledgement, error) {
	// language=postgresql
	query := `
		SELECT id, accepted, receipt_time, rejected_reason
		FROM publication_ack
		WHERE version_id = $1;
	`

	row := tx.QueryRow(ctx, query, versionID)

	var ack Acknowledgement
	var rejectedReason *string
	err := row.Scan(&ack.ID, &ack.Accepted, &ack.Time, &rejectedReason)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if rejectedReason != nil {
		ack.RejectedReason = *rejectedReason
	}

	return &ack, nil
}

func CreatePublicationAcknowledgement(ctx context.Context, tx pgx.Tx, pubID, versionID string, t time.Time, accept bool,
	reason string, idempotent bool) error {

	// check that the version exists and is associated with the specified publication
	actualPubID, err := GetPublicationIDForVersion(ctx, tx, versionID)
	if err != nil {
		return err
	}
	if actualPubID != pubID {
		return status.Error(codes.NotFound, "version not found for publication")
	}

	if idempotent {
		// before attempting an update, check if it was already acknowledged
		ack, err := GetAcknowledgement(ctx, tx, versionID)
		if err != nil {
			return err
		}

		if ack != nil {
			if ack.Accepted != accept {
				return errors.New("conflicting acknowledgement already exists")
			} else {
				// an acknowledgement of the same kind already exists, so we don't need to acknowledge again
				return nil
			}
		}
	}

	// language=postgresql
	query := `
		INSERT INTO publication_ack (id, version_id, accepted, rejected_reason, receipt_time) 
		VALUES (DEFAULT, $1, $2, $3, $4);
    `

	_, err = tx.Exec(ctx, query, versionID, accept, reason, t)
	return err
}

type publicationVersionRow struct {
	PublicationID  string
	AudienceName   *string
	VersionID      string
	Body           []byte
	PublishTime    time.Time
	MediaType      *string
	Accepted       *bool
	ReceiptTime    *time.Time
	RejectedReason *string
}

func publicationVersionRowToProto(row publicationVersionRow) *publicationpb.Publication {
	receipt := publicationpb.Publication_Audience_NO_SIGNAL
	if row.Accepted != nil {
		if *row.Accepted {
			receipt = publicationpb.Publication_Audience_ACCEPTED
		} else {
			receipt = publicationpb.Publication_Audience_REJECTED
		}
	}

	audience := &publicationpb.Publication_Audience{Receipt: receipt}
	if row.AudienceName != nil {
		audience.Name = *row.AudienceName
	}
	if row.ReceiptTime != nil {
		audience.ReceiptTime = timestamppb.New(*row.ReceiptTime)
	}
	if row.RejectedReason != nil {
		audience.ReceiptRejectedReason = *row.RejectedReason
	}

	pub := &publicationpb.Publication{
		Id:          row.PublicationID,
		Version:     row.VersionID,
		Body:        row.Body,
		Audience:    audience,
		PublishTime: timestamppb.New(row.PublishTime),
	}
	if row.MediaType != nil {
		pub.MediaType = *row.MediaType
	}

	return pub
}
