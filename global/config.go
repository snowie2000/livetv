package global

import (
	"encoding/base64"

	"github.com/jinzhu/gorm"
	"github.com/snowie2000/livetv/model"
	"golang.org/x/crypto/scrypt"
)

func strongKey(key string) []byte {
	//make a key strong by using scrypt
	dk, _ := scrypt.Key([]byte(key), []byte("dasdADD123@#as84373^!$*&!#$1#12#"), 16384, 8, 1, 32)
	return dk
}

func GetConfig(key string) (string, error) {
	if confValue, ok := ConfigCache.Load(key); ok {
		return confValue, nil
	} else {
		var value model.Config
		err := DB.Where("name = ?", key).First(&value).Error
		if err != nil {
			if gorm.IsRecordNotFoundError(err) {
				return "", ErrConfigNotFound
			} else {
				return "", err
			}
		} else {
			ConfigCache.Store(key, value.Data)
			return value.Data, nil
		}
	}
}

var strongSecret string = ""
var strongLiveSecret string = ""

func GetSecretToken() string {
	if strongSecret == "" {
		secret, _ := GetConfig("secret")
		if secret == "" {
			return ""
		}
		derived := strongKey(secret)
		strongSecret = string([]rune(base64.URLEncoding.EncodeToString(derived))[1:10])
	}
	return strongSecret
}

func GetLiveToken() string {
	if strongLiveSecret == "" {
		secret, _ := GetConfig("secret")
		if secret == "" {
			return ""
		}
		derived := strongKey(secret + "_live")
		strongLiveSecret = string([]rune(base64.URLEncoding.EncodeToString(derived))[1:10])
	}
	return strongLiveSecret
}

func ClearSecretToken() {
	strongSecret = ""
	strongLiveSecret = ""
	ChannelCache.Clear()
}

func SetConfig(key, value string) error {
	data := model.Config{Name: key, Data: value}
	err := DB.Save(&data).Error
	if err == nil {
		ConfigCache.Store(key, value)
	}
	return err
}
