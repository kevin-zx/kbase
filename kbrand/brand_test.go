package kbrand

import (
	"fmt"
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
		// 阿飞和巴弟（Alfie&Buddy）
		{
			name: "阿飞和巴弟（Alfie&Buddy）",
			args: args{
				raw: "阿飞和巴弟（Alfie&Buddy）",
			},
			want: Brand{
				Raw: "阿飞和巴弟（Alfie&Buddy）",
				EN:  "Alfie&Buddy",
				CN:  "阿飞和巴弟",
			},
		},
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
		// 3M
		{
			name: "3M",
			args: args{
				raw: "3M",
			},
			want: Brand{
				Raw: "3M",
				EN:  "3M",
				CN:  "",
			},
		},
		// 8H
		{
			name: "8H",
			args: args{
				raw: "8H",
			},
			want: Brand{
				Raw: "8H",
				EN:  "8H",
				CN:  "",
			},
		},
		// 第一森林（First Forest）
		{
			name: "第一森林（First Forest）",
			args: args{
				raw: "第一森林（First Forest）",
			},
			want: Brand{
				Raw: "第一森林（First Forest）",
				EN:  "First Forest",
				CN:  "第一森林",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseBrand(tt.args.raw)
			fmt.Printf("got: %+v\n", got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseBrand() = %v, want %v", got, tt.want)
			}
		})
	}
}
