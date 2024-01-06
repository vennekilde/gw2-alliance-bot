package orm

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/vennekilde/gw2apidb/internal/config"
)

var db *gorm.DB

func DB() *gorm.DB {
	if db == nil {
		conStr := fmt.Sprintf(
			"host=%s port=%d user=%s dbname=%s password=%s sslmode=disable",
			config.Config().PostgresHost,
			config.Config().PostgresPort,
			config.Config().PostgresUser,
			config.Config().PostgresDatabase,
			config.Config().PostgresPassword)
		var err error
		db, err = gorm.Open("postgres", conStr)

		if err != nil {
			panic(err)
		}
	}
	return db
}
