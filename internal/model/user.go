package model

import "time"

type User struct {
	ID        int       `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`         // 账号名
	Nickname  string    `db:"nickname" json:"nickname"` // 展示昵称
	Email     string    `db:"email" json:"email"`
	Password  string    `db:"password" json:"-"`
	Age       int       `db:"age" json:"age"`
	Gender    int       `db:"gender" json:"gender"`
	Avatar    string    `db:"avatar" json:"avatar"`
	Status    int       `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
