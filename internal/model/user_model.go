package model

import "time"

type User struct {
	ID        uint      `gorm:"column:id;primarykey;autoIncrement"`
	Username  string    `gorm:"column:username;unique"`
	Password  string    `gorm:"column:password;not null"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}
