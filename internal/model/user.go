package model

type User struct {
	ID           uint   `gorm:"primarykey;autoIncrement"`
	Username     string `gorm:"column:username;unique"`
	PasswordHash string `gorm:"column:password;NOT NULL"`
}
