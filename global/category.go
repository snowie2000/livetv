package global

import (
	"github.com/snowie2000/livetv/model"
	"strings"
)

func GetAllCategories() []string {
	var channels []model.Channel
	var categories []string
	err := DB.Group("category").Find(&channels).Error
	if err == nil {
		for _, v := range channels {
			c := strings.TrimSpace(v.Category)
			if c != "" {
				categories = append(categories, v.Category)
			}
		}
	}
	return categories
}
