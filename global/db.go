package global

import (
	"fmt"
	"strings"
	"sync"

	"github.com/jinzhu/gorm"
	"github.com/snowie2000/livetv/model"
	"github.com/snowie2000/livetv/util"
	_ "modernc.org/sqlite"
)

var (
	DB        *gorm.DB
	DBWriteMu sync.Mutex
)

func InitDB(filepath string) (err error) {
	dsn := filepath
	if !strings.Contains(filepath, "?") {
		dsn = fmt.Sprintf("%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(1)", filepath)
	}
	DB, err = gorm.Open("sqlite", dsn)
	if err != nil {
		return err
	}

	sqlDB := DB.DB()
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	// Ensure WAL mode and busy timeout are active
	_ = DB.Exec("PRAGMA journal_mode = WAL;").Error
	_ = DB.Exec("PRAGMA busy_timeout = 10000;").Error
	_ = DB.Exec("PRAGMA synchronous = NORMAL;").Error
	_ = DB.Exec("PRAGMA foreign_keys = ON;").Error

	err = DB.AutoMigrate(&model.Config{}, &model.Channel{}).Error
	if err != nil {
		return err
	}
	// update old parsers to their new names
	DB.Model(&model.Channel{}).Where("parser IN (?)", []string{"httpRedirect", "direct"}).Update("parser", "http")

	// set default value for configs
	for key, valueDefault := range defaultConfigValue {
		var valueInDB model.Config
		err = DB.Where("name = ?", key).First(&valueInDB).Error
		if err != nil {
			if gorm.IsRecordNotFoundError(err) {
				ConfigCache.Store(key, valueDefault)
			} else {
				return err
			}
		} else {
			ConfigCache.Store(key, valueInDB.Data)
		}
	}

	// ensure a strong secret is initialized
	var secretInDB model.Config
	err = DB.Where("name = ?", "secret").First(&secretInDB).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			_ = SetConfig("secret", util.RandString(16))
		} else {
			return err
		}
	} else if secretInDB.Data == "" {
		_ = SetConfig("secret", util.RandString(16))
	} else {
		ConfigCache.Store("secret", secretInDB.Data)
	}
	return nil
}

func init() {
	// use sqlite3 dialect for sqlite
	if dialect, ok := gorm.GetDialect("sqlite3"); ok {
		gorm.RegisterDialect("sqlite", dialect)
	}
}
