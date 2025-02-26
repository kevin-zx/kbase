package kproductmodel

import (
	"regexp"
	"strings"
)

type KproductModelExtractor struct {
	brands   []string
	excludes []string
	preModel []string
}

func NewKproductModelExtractor(brands []string, excludes []string, preModels []string) *KproductModelExtractor {

	for i, b := range brands {
		brands[i] = strings.ToLower(b)
	}

	for i, e := range excludes {
		excludes[i] = strings.ToLower(e)
	}

	for i, p := range preModels {
		preModels[i] = strings.ToLower(p)
	}

	return &KproductModelExtractor{
		brands: brands, excludes: excludes,
		preModel: preModels,
	}
}

var modelRegex = regexp.MustCompile(`[A-Za-z]+[A-Za-z0-9-_ ]*[A-Za-z0-9]+`)

func (k *KproductModelExtractor) ExtractModel(s string) (string, bool) {
	s = k.clean(s)
	// 包含预定义型号的情况
	for _, preModel := range k.preModel {
		if strings.Contains(s, preModel) {
			return preModel, true
		}
	}
	// 使用正则表达式查找所有匹配的部分，并返回第一个找到的型号部分（假设只有一个）
	if match := modelRegex.FindString(s); match != "" {
		return k.cleanModel(match), true
	}

	// 如果没有找到任何匹配，返回空字符串和 false
	return "", false
}

// 如果 model 开头是符号，例如“-”，则将其删除
// 使用 正则
var reLeading = regexp.MustCompile(`^[^\p{L}\p{N}]+`)
var reTrailing = regexp.MustCompile(`[^\p{L}\p{N}]+$`)

func (k *KproductModelExtractor) cleanModel(model string) string {
	model = reLeading.ReplaceAllString(model, "")
	model = reTrailing.ReplaceAllString(model, "")
	return model
}

func (k *KproductModelExtractor) clean(s string) string {
	s = strings.ToLower(s)
	for _, exclude := range k.excludes {
		s = strings.ReplaceAll(s, exclude, "")
	}
	for _, b := range k.brands {
		s = strings.ReplaceAll(s, b, "")
	}

	return s
}
