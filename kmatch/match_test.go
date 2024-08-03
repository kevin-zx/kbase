package kmatch

import "testing"

func Test_kMatcher_Match(t *testing.T) {
	type args struct {
		txt string
	}
	tests := []struct {
		name string
		m    *kMatcher
		args args
		want bool
	}{
		{
			name: "test1",
			m: &kMatcher{
				km: KMatch{
					Matches:  []string{"a", "b"},
					Musts:    []string{"c", "d"},
					Excludes: []string{"e", "f"},
				},
				ignoreCase: false,
			},
			args: args{
				txt: "cdbss",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.Match(tt.args.txt); got != tt.want {
				t.Errorf("kMatcher.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}
