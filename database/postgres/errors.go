package postgres

import (
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func IsUniqueViolation(err error, constraint string) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)

	return ok &&
		pgErr.Code == pgerrcode.UniqueViolation &&
		(constraint == "" || pgErr.ConstraintName == constraint)
}
