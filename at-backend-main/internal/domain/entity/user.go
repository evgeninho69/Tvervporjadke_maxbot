package entity

import "time"

type User struct {
	ID           uint64
	Email        string
	RegisteredAt time.Time
}

type BlockedUser struct {
	ID        uint64
	Email     string
	UID       uint64
	Reason    string
	BlockedAt time.Time
	BlockedBy uint64
}
