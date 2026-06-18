/**
@Time : 2026/01/16 09:16
@Author: FangYao( 方少、)
@Description:主控制器层
@Email: fy20030315@163.com
*/

package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zht "github.com/go-playground/validator/v10/translations/zh"
	"go-nvr/pkg/plugins"
)

var (
	validate = validator.New()
	trans    ut.Translator

	// 设备服务
	Device = &DeviceController{}
	// 代理
	Proxy = &ProxyController{}
)

func init() {
	uni := ut.New(zh.New())                              // 中文
	trans, _ = uni.GetTranslator("zh")                   // 将校验错误信息翻译成中文
	_ = zht.RegisterDefaultTranslations(validate, trans) // 注册默认中文翻译

}

// 通用接口处理封装
func Run(c *gin.Context, req interface{}, fn func() (interface{}, interface{})) {
	var err error
	// 绑定请求参数到结构体
	err = c.Bind(req)
	if err != nil {
		fmt.Println("err", err)
		plugins.HttpErr(c, plugins.NewValidatorError(err), nil)
		return
	}
	// 校验请求参数
	err = validate.Struct(req)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			plugins.HttpErr(c, plugins.NewValidatorError(fmt.Errorf(err.Translate(trans))), nil)
			return
		}
	}

	// 执行业务逻辑
	data, err1 := fn()
	if err1 != nil {
		// 处理业务逻辑错误
		plugins.HttpErr(c, plugins.ReloadErr(err1), data)
		return
	}
	// 返回响应
	plugins.HttpSuccess(c, data)
}
