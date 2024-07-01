package kbrand

import (
	"reflect"
	"testing"
)

func TestParseBrand(t *testing.T) {
	type args struct {
		raw string
	}
	tests := []struct {
		name string
		args args
		want Brand
	}{
		// "Apple 苹果",
		{
			name: "Apple 苹果",
			args: args{
				raw: "Apple 苹果",
			},
			want: Brand{
				Raw: "Apple 苹果",
				EN:  "Apple",
				CN:  "苹果",
			},
		},
		// "南极人（Nanjiren）",
		{
			name: "南极人（Nanjiren）",
			args: args{
				raw: "南极人（Nanjiren）",
			},
			want: Brand{
				Raw: "南极人（Nanjiren）",
				EN:  "Nanjiren",
				CN:  "南极人",
			},
		},
		// "MUJI",
		{
			name: "MUJI",
			args: args{
				raw: "MUJI",
			},
			want: Brand{
				Raw: "MUJI",
				EN:  "MUJI",
				CN:  "",
			},
		},
		// "网易严选",
		{
			name: "网易严选",
			args: args{
				raw: "网易严选",
			},
			want: Brand{
				Raw: "网易严选",
				EN:  "",
				CN:  "网易严选",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseBrand(tt.args.raw); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseBrand() = %v, want %v", got, tt.want)
			}
		})
	}
}
