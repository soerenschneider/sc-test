package pkg

import "testing"

func TestIsAsciiNumeric(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "happy path",
			args: args{
				s: "1234567890",
			},
			want: true,
		},
		{
			name: "edge case zero",
			args: args{
				s: "0",
			},
			want: true,
		},
		{
			name: "edge case valid number with space",
			args: args{
				s: "234 ",
			},
			want: false,
		},
		{
			name: "letter in number",
			args: args{
				s: "1234a",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAsciiNumeric(tt.args.s); got != tt.want {
				t.Errorf("IsAsciiNumeric() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOtpValidation(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "happy path: 6 digits",
			args: args{
				s: "123456",
			},
			wantErr: false,
		},
		{
			name: "happy path: 8 digits",
			args: args{
				s: "12345678",
			},
			wantErr: false,
		},
		{
			name: "less than 6 digits",
			args: args{
				s: "12345",
			},
			wantErr: true,
		},
		{
			name: "more than 8 digits",
			args: args{
				s: "123456789",
			},
			wantErr: true,
		},
		{
			name: "7 digits",
			args: args{
				s: "1234567",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := OtpValidation(tt.args.s); (err != nil) != tt.wantErr {
				t.Errorf("OtpValidation() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
