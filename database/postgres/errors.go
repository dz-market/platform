package postgres

import (
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func IsUniqueViolation(err error, constraint string) bool {
	return isViolation(err, pgerrcode.UniqueViolation, constraint)
}

func isViolation(err error, code, constraint string) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)

	return ok && pgErr.Code == code && (constraint == "" || pgErr.ConstraintName == constraint)
}
