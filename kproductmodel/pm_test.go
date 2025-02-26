package kproductmodel

import (
	"testing"
)

func TestExtractModel(t *testing.T) {
	// "保友金豪b2雄鹰人体工学椅",
	// "西昊m59as人体工学椅",
	// 100元级别人体工学椅
	tests := []struct {
		name     string
		input    string
		expected string
		found    bool
	}{
		{"ValidModel1", "BrandX Model123", "model123", true},
		{"ValidModel2", "BrandY-Model456", "model456", true},
		{"ValidModel3", "BrandZ Model_789", "model_789", true},
		{"ExcludedModel", "BrandX ExcludedModel123", "model123", true},
		{"ExcludedModel2", "保友金豪b2雄鹰人体工学椅", "金豪b2", true},
		{"ExcludedModel3", "西昊m59as人体工学椅", "m59as", true},
		{"ExcludedModel4", "100元级别人体工学椅", "", false},
		{"ExcludedModel5", "永艺撑腰椅沃克PRO 人体工学电脑椅 家用办公电竞椅子 透气可躺带脚托", "沃克pro", true},
	}

	brands := []string{"BrandX", "BrandY", "BrandZ"}
	excludes := []string{"Excluded"}
	premodels := []string{"沃克PRO", "金豪b2"}

	extractor := NewKproductModelExtractor(brands, excludes, premodels)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, found := extractor.ExtractModel(tt.input)
			if result != tt.expected || found != tt.found {
				t.Errorf("ExtractModel(%s) = (%s, %v), want (%s, %v)", tt.input, result, found, tt.expected, tt.found)
			}
		})
	}
}
