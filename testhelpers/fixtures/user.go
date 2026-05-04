package fixtures

import (
	"fmt"

	"github.com/Arup3201/torb/models"
	"github.com/google/uuid"
)

func RandomUserRow() models.User {
	uId := uuid.NewString()

	return models.User{
		ID:       uId,
		Username: "user" + uId,
		Email:    "user" + uId + "@test.com",
	}
}

func (f *Fixtures) InsertUser(u models.User) string {
	if f.db != nil {
		if err := f.db.WithContext(f.ctx).Create(&u).Error; err != nil {
			panic(fmt.Sprintf("insert user fixture failed: %v", err))
		}
		return u.ID
	}
	return ""
}
