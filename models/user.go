package models

import "time"

type User struct {
	ID        int64
	createdAt time.Time
}
