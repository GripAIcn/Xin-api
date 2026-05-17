package model

type User struct {
	ID       uint   `json:"id" gorm:"column:id;primarykey;autoIncrement"`
	Username string `json:"username" gorm:"column:username;unique" validate:"max=12"`
	Password string `json:"password" gorm:"column:password;NOT NULL" validate:"min=8,max=20"`
}
