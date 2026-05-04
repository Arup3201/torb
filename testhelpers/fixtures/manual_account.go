package fixtures

import (
	"fmt"

	"github.com/Arup3201/torb/models"
)

func GetManualAccount(userID, email, password string) models.ManualAccount {

	return models.ManualAccount{
		UserID:        userID,
		Email:         email,
		PasswordHash:  []byte(password),
		EmailVerified: false,
	}
}

func (f *Fixtures) InsertManualAccount(account models.ManualAccount) {
	if err := f.db.WithContext(f.ctx).Create(&account).Error; err != nil {
		panic(fmt.Sprintf("insert manual account fixture failed: %v", err))
	}
}
