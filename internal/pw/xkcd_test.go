package pw

import "testing"

func Test_replaceNthOccurrence(t *testing.T) {
	type args struct {
		s   string
		old string
		new string
		n   int
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "replace 1st occurrence",
			args: args{"one two three two four", "two", "TWO", 1},
			want: "one TWO three two four",
		},
		{
			name: "replace 2nd occurrence",
			args: args{"one two three two four", "two", "TWO", 2},
			want: "one two three TWO four",
		},
		{
			name: "replace 3rd occurrence",
			args: args{"one two three two four two", "two", "TWO", 3},
			want: "one two three two four TWO",
		},
		{
			name: "n greater than occurrences",
			args: args{"one two three", "two", "TWO", 5},
			want: "one two three", // unchanged
		},
		{
			name:    "n is zero",
			args:    args{"one two three", "two", "TWO", 0},
			want:    "one two three", // invalid n, unchanged
			wantErr: true,
		},
		{
			name:    "n is negative",
			args:    args{"one two three", "two", "TWO", -1},
			want:    "one two three", // invalid n, unchanged
			wantErr: true,
		},
		{
			name: "old string not found",
			args: args{"one two three", "four", "FOUR", 1},
			want: "one two three", // unchanged
		},
		{
			name: "empty string input",
			args: args{"", "a", "b", 1},
			want: "", // unchanged
		},
		{
			name: "replace with empty string",
			args: args{"one two three two", "two", "", 2},
			want: "one two three ", // second "two" removed
		},
		{
			name: "replace substring longer than one char",
			args: args{"ababab", "ab", "XY", 2},
			want: "abXYab",
		},
		{
			name: "replace last occurrence",
			args: args{"repeat repeat repeat", "repeat", "R", 3},
			want: "repeat repeat R",
		},
		{
			name: "replace only occurrence when n=1",
			args: args{"hello", "hello", "hi", 1},
			want: "hi",
		},
		{
			name:    "don't replace anything",
			args:    args{"example passphrase very secure", " ", "-", 0},
			want:    "example passphrase very secure",
			wantErr: true,
		},
		{
			name: "replace 1st space",
			args: args{"example passphrase very secure", " ", "-", 1},
			want: "example-passphrase very secure",
		},
		{
			name: "replace 2nd space",
			args: args{"example passphrase very secure", " ", "-", 2},
			want: "example passphrase-very secure",
		},
		{
			name: "replace 3rd space",
			args: args{"example passphrase very secure", " ", "-", 3},
			want: "example passphrase very-secure",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := replaceNthOccurrence(tt.args.s, tt.args.old, tt.args.new, tt.args.n)
			if (err != nil) != tt.wantErr {
				t.Errorf("replaceNthOccurrence() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("replaceNthOccurrence() got = %v, want %v", got, tt.want)
			}
		})
	}
}
