package recaptcha

import (
	"github.com/mojocn/base64Captcha"
)

var DefaultCaptcha = &CaptchaTool{
	store: base64Captcha.DefaultMemStore,
}

type CaptchaTool struct {
	store base64Captcha.Store
}

type CaptchaData struct {
	CaptchaId string `json:"captcha_id"`
	Data      string `json:"data"`
	Answer    string `json:"answer"`
}

var digitDriver = base64Captcha.DriverDigit{
	Height:   50,
	Width:    130,
	Length:   4,   // captcha length
	MaxSkew:  0.5, // italic strength
	DotCount: 1,   // background noises
}

func (c *CaptchaTool) GenerateCaptcha() (*CaptchaData, error) {
	code := base64Captcha.NewCaptcha(&digitDriver, c.store)
	id, b64s, _, err := code.Generate()
	if err != nil {
		return nil, err
	}
	return &CaptchaData{
		CaptchaId: id,
		Data:      b64s,
	}, nil
}
func (c *CaptchaTool) Verify(data *CaptchaData) bool {
	return c.store.Verify(data.CaptchaId, data.Answer, true)
}
