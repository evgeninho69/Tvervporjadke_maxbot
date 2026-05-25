package roles

type Role string

const (
	Admin      Role = "operator"
	SuperAdmin Role = "super-admin"
	User       Role = "user"
)

func (r Role) ToString() string {
	return string(r)
}

func IsAdmin(role Role) bool {
	switch role {
	case Admin, SuperAdmin:
		return true
	default:
		return false
	}
}

func IsSuperAdmin(role Role) bool {
	switch role {
	case SuperAdmin:
		return true
	default:
		return false
	}
}