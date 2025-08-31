package domain

import "github.com/google/uuid"

type Cron struct {
	ID uuid.UUID `json:"id"`
}
