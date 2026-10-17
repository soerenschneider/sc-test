package agent

import "testing"

func TestAddDefaultProtoAndPort(t *testing.T) {
	type args struct {
		server      string
		useHttps    bool
		defaultPort uint16
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "nothing to do",
			args: args{
				server:      "https://server.tld:443",
				useHttps:    true,
				defaultPort: 8080,
			},
			want: "https://server.tld:443",
		},
		{
			name: "add port",
			args: args{
				server:      "https://server.tld",
				useHttps:    true,
				defaultPort: 8080,
			},
			want: "https://server.tld:8080",
		},
		{
			name: "add protocol",
			args: args{
				server:      "server.tld:8080",
				useHttps:    true,
				defaultPort: 443,
			},
			want: "https://server.tld:8080",
		},
		{
			name: "add protocol and port",
			args: args{
				server:      "server.tld",
				useHttps:    true,
				defaultPort: 443,
			},
			want: "https://server.tld:443",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AddDefaultProtoAndPort(tt.args.server, tt.args.useHttps, tt.args.defaultPort); got != tt.want {
				t.Errorf("AddDefaultProtoAndPort() = %v, want %v", got, tt.want)
			}
		})
	}
}
