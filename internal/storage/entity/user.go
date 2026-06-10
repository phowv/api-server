package entity

type User struct {
	UserId      				int       `gorm:"column:user_id;primaryKey"`
	Login        				string    `gorm:"column:login"`
	HashPassword				string		`gorm:"column:hash_password"`
	Email								string		`gorm:"column:email"`
	Role								string		`gorm:"column:role"`
	Description  				string    `gorm:"column:description"`
}

func (User) TableName() string {
	return "users.users"
}
