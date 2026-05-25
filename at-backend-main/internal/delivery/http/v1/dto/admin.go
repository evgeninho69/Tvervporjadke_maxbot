package dto

import (
	_ "encoding/json"
)

type AdminPrivilegesResponse struct {
	IsAdmin      bool `json:"is_admin"`
	IsSuperAdmin bool `json:"is_super_admin"`
}

type AdminResponse struct {
	Email        string `json:"email"`
	IsSuperAdmin bool   `json:"is_super_admin"`
}
