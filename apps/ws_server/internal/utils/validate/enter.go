package validate

// File: utils/validate/enter.go
// Description: 参数校验模块，提供基于validator的参数校验及中文错误翻译功能

import (
	"reflect"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/zh"
	"github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
)

// trans 全局翻译器实例，用于将验证错误信息转换为中文
var trans ut.Translator

// init 初始化验证器翻译配置，注册中文翻译及自定义字段标签解析
func init() {
	// 创建中文翻译器实例
	uni := ut.New(zh.New())
	trans, _ = uni.GetTranslator("zh")

	// 获取Gin绑定器中的validator实例并注册中文翻译
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if ok {
		_ = zh_translations.RegisterDefaultTranslations(v, trans)
	}

	// 注册自定义字段标签解析函数，优先使用label标签作为字段名展示
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		label := field.Tag.Get("label")
		if label == "" {
			return field.Name // 无label标签时使用结构体字段名
		}
		return label
	})
}

// ValidateError 将validator验证错误转换为中文提示信息
func ValidateError(err error) string {
	// 类型断言判断是否为validator验证错误
	errs, ok := err.(validator.ValidationErrors)
	if !ok {
		return err.Error() // 非验证错误直接返回原始错误信息
	}

	// 遍历所有验证错误并转换为中文
	var list []string
	for _, e := range errs {
		list = append(list, e.Translate(trans))
	}

	// 拼接多个错误信息为单个字符串返回
	return strings.Join(list, ";")
}
