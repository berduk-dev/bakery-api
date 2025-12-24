package model

import "time"

type Prize struct {
	ID        int64      `json:"id"`
	Code      string     `json:"code"`
	Prize     string     `json:"prize"`
	CreatedAt *time.Time `json:"created_at"`
	UsedAt    *time.Time `json:"used_at"`
}

type User struct {
	TelegramID int64
	Phone      *string
	CreatedAt  time.Time
}
