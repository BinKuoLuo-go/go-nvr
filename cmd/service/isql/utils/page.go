/**
@Time : 2026/03/10 14:42
@Author: FangYao( 方少、)
@Description: 分页器
@Email: fy20030315@163.com
*/

package utils

type PageOption struct {
	PageNum  int `json:"pageNum"`
	PageSize int `json:"pageSize"`
}

var defaultOptions *PageOption

func init() {
	// 默认取 第 1 页的 10 条记录
	defaultOptions = &PageOption{
		PageNum:  0,
		PageSize: 10,
	}
}

// 分页
func NewPageOption(pageNum, pageSize int) *PageOption {
	if pageSize <= 0 || pageNum > 1000 || pageNum < 0 {
		return defaultOptions
	}
	pNum := (pageNum - 1) * pageSize
	return &PageOption{
		PageNum:  pNum,
		PageSize: pageSize,
	}
}
