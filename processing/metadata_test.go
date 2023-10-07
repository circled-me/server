package processing

import "testing"

func Test_getTimeOffsetFrom(t *testing.T) {
	t9 := 9 * 3600
	t95 := 9*3600 + 1800
	m95 := -(9*3600 + 1800)
	t0 := 0
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want *int
	}{
		{
			"err",
			args{"asas"},
			nil,
		},
		{
			"+09:00",
			args{"+09:00"},
			&t9,
		},
		{
			"+00:00",
			args{"+00:00"},
			&t0,
		},
		{
			"+09:30",
			args{"+09:30"},
			&t95,
		},
		{
			"-09:30",
			args{"-09:30"},
			&m95,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTimeOffsetFrom(tt.args.s)
			if got == tt.want || (got != nil && tt.want != nil && *got == *tt.want) {
				return // ok
			}
			t.Errorf("getTimeOffsetFrom() = %v, want %v", got, tt.want)
		})
	}
}
