package pgxhub

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type Enrollment struct {
	Name        string
	Description string
	Address     string
	Cert        []byte
}

const rowFields = "address, name, description, cert"

func SelectEnrollment(ctx context.Context, tx pgx.Tx, address string) (en Enrollment, err error) {
	// language=postgresql
	query := `
		SELECT ` + rowFields + `
		FROM enrollment
		WHERE address = $1;
    `

	row := tx.QueryRow(ctx, query, address)
	err = scanRow(row, &en)
	return
}

func InsertEnrollment(ctx context.Context, tx pgx.Tx, en Enrollment) error {
	// language=postgresql
	query := `
		INSERT INTO enrollment (address, name, description, cert) 
		VALUES ($1, $2, $3, $4);
	`

	var nameNull, descNull *string
	if en.Name != "" {
		nameNull = &en.Name
	}
	if en.Description != "" {
		descNull = &en.Description
	}

	_, err := tx.Exec(ctx, query, en.Address, nameNull, descNull, en.Cert)
	return err
}

func UpdateEnrollment(ctx context.Context, tx pgx.Tx, en Enrollment) error {
	// language=postgresql
	query := `
		UPDATE enrollment SET name=$2, description=$3, cert=$4
		WHERE address=$1;
	`

	var nameNull, descNull *string
	if en.Name != "" {
		nameNull = &en.Name
	}
	if en.Description != "" {
		descNull = &en.Description
	}

	_, err := tx.Exec(ctx, query, en.Address, nameNull, descNull, en.Cert)
	return err
}

func DeleteEnrollment(ctx context.Context, tx pgx.Tx, address string) error {
	// language=postgresql
	query := `DELETE FROM enrollment WHERE address=$1`
	_, err := tx.Exec(ctx, query, address)
	return err
}

func SelectEnrollments(ctx context.Context, tx pgx.Tx) ([]Enrollment, error) {
	// language=postgresql
	query := `
		SELECT ` + rowFields + `
		FROM enrollment
		ORDER BY address;
	`

	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var enrollments []Enrollment
	for rows.Next() {
		var en Enrollment
		err = scanRow(rows, &en)
		if err != nil {
			return nil, err
		}

		enrollments = append(enrollments, en)
	}
	return enrollments, nil
}

func scanRow(row pgx.Row, dst *Enrollment) error {
	var nameNull, descNull *string
	err := row.Scan(&dst.Address, &nameNull, &descNull, &dst.Cert)
	if err != nil {
		return err
	}
	if nameNull != nil {
		dst.Name = *nameNull
	}
	if descNull != nil {
		dst.Description = *descNull
	}
	return nil
}
