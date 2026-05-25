package models

type AdminData struct {
	UID          uint64 `db:"uid"`
	Email        string `db:"email"`
	IsSuperAdmin bool   `db:"is_super_admin"`
}
