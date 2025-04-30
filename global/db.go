package global

import (
	"github.com/jinzhu/gorm"
	"github.com/snowie2000/livetv/model"
	_ "modernc.org/sqlite"
)

var DB *gorm.DB

func InitDB(filepath string) (err error) {
	DB, err = gorm.Open("sqlite", filepath)
	if err != nil {
		return err
	}
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
	return nil
}

func init() {
	// use sqlite3 dialect for sqlite
	if dialect, ok := gorm.GetDialect("sqlite3"); ok {
		gorm.RegisterDialect("sqlite", dialect)
	}
}
