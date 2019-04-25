package users

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres" // Dialect import
	"github.com/lbryio/lbryweb.go/db"
)

// User is a thin model containing basic data about lbryweb user.
// The majority of user data is stored in internal-apis, referenced by AuthToken
type User struct {
	gorm.Model
	CreatedAt    time.Time `boil:"created_at" json:"created_at"`
	AuthToken    string    `boil:"auth_token" json:"auth_token"`
	SDKAccountID string    `boil:"sdk_account_id" json:"sdk_account_id"`
}

// AutoMigrate migrates user table
func AutoMigrate() {
	db.DB.AutoMigrate(&User{})
}

// GetRecordByToken retrieves user record by token
func GetRecordByToken(token string) (u User) {
	db.DB.First(&u, "auth_token = ?", token)
	return u
}

// CreateRecord saves user record to the database
func CreateRecord(accountID, token string) error {
	u := User{}
	if GetRecordByToken(token) != u {
		return fmt.Errorf("user %v already exists", token)
	}
	db.DB.Create(&User{AuthToken: token, SDKAccountID: accountID})
	return nil
}
