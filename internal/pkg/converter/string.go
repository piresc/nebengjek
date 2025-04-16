package converter

import (
	"github.com/google/uuid"
)

func StrToUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}

func UUIDToStr(id uuid.UUID) string {
	return id.String()
}
