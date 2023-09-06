package osc

import "testing"

func Test_byteToFloat32(t *testing.T) {
	type args struct {
		b []byte
	}
	tests := []struct {
		name string
		args args
		want float32
	}{
		{
			name: "",
			args: args{
				[]byte{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := byteToFloat32(tt.args.b); got != tt.want {
				t.Errorf("byteToFloat32() = %v, want %v", got, tt.want)
			}
		})
	}
}
