package models

import "database/sql"

type User struct {
	ID            int64
	CitizenshipID sql.NullString
	DocumentFiles sql.NullString
	AddressFiles  sql.NullString
}
