package types

import (
	"encoding/json"
)

type ClaimStatus string

const (
	ClaimStatusPending   ClaimStatus = "pending"   // в обработке
	ClaimStatusAccepted  ClaimStatus = "accepted"  // принята в работу
	ClaimStatusCompleted ClaimStatus = "completed" // выполнена
	ClaimStatusDeclined  ClaimStatus = "declined"  // отклонена
	ClaimStatusUnknown   ClaimStatus = "unknown"   // для обработки некорректного ввода
)

func (cs ClaimStatus) ToString() string {
	return string(cs)
}

func (cs *ClaimStatus) UnmarshalJSON(b []byte) error {
	var s string

	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	switch s {
	case "pending":
		*cs = ClaimStatusPending
	case "accepted":
		*cs = ClaimStatusAccepted
	case "completed":
		*cs = ClaimStatusCompleted
	case "declined":
		*cs = ClaimStatusDeclined
	default:
		*cs = ClaimStatusUnknown
	}

	return nil
}
