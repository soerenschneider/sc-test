package internal

import (
	"reflect"
	"testing"
)

func Test_getCommandHierarchy(t *testing.T) {
	type args struct {
		cmd string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "happy case",
			args: args{
				cmd: "a-b-c",
			},
			want: []string{
				"a-b-c",
				"a-b",
				"a",
			},
		},
		{
			name: "edge case empty",
			args: args{
				cmd: "",
			},
			want: []string{
				"",
			},
		},
		{
			name: "edge case",
			args: args{
				cmd: "a",
			},
			want: []string{
				"a",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getCommandHierarchy(tt.args.cmd); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getCommandHierarchy() = %v, want %v", got, tt.want)
			}
		})
	}
}
