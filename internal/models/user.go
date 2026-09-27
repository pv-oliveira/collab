package models

import "time"

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // hash bcrypt: nunca sai em respostas da API
	CreatedAt time.Time `json:"created_at"`
}
