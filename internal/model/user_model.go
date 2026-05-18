package model

type User struct {
	ID       uint   `gorm:"column:id;primarykey;autoIncrement"`
	Username string `gorm:"column:username;unique"`
	Password string `gorm:"column:password;NOT NULL"`
}
