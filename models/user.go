package models

import "database/sql"

type User struct {
	ID             int64
	CitizenshipID  sql.NullString
	DocumentFiles  sql.NullString
	AddressFiles   sql.NullString
	Phone          sql.NullString
	Email          sql.NullString
	FirstName      sql.NullString
	LastName       sql.NullString
	Patronymic     sql.NullString
	DocumentNumber sql.NullString
}
