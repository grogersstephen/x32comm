package main

import "testing"

func Test_getFaderPath(t *testing.T) {
	type args struct {
		ch int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should pass; 33",
			args: args{
				ch: 33,
			},
			want: `\ch\33\mix\fader`,
		},
		{
			name: "should pass; 01",
			args: args{
				ch: 1,
			},
			want: `\ch\01\mix\fader`,
		},
		{
			name: "should pass; even though this is not a valid channel",
			args: args{
				ch: 100,
			},
			want: `\ch\100\mix\fader`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getFaderPath(tt.args.ch); got != tt.want {
				t.Errorf("getFaderPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
