package model

type User struct {
	ID       uint   `gorm:"column:id;primarykey;autoIncrement"`
	Username string `gorm:"column:username;unique" validate:"max=12"`
	Password string `gorm:"column:password;NOT NULL" validate:"min=8,max=20"`
}
