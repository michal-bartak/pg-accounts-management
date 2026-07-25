package pg

import "testing"

func TestCleanConnectError(t *testing.T) {
	seg := "127.0.0.1:5002 (127.0.0.1): dial error: dial tcp 127.0.0.1:5002: connect: connection refused"
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "prefer-mode duplicated dial error is collapsed",
			in:   "failed to connect to `user=postgres database=replica_dwh`: " + seg + " " + seg,
			want: "failed to connect to `user=postgres database=replica_dwh`: " + seg,
		},
		{
			name: "newline/tab separators are normalized then collapsed",
			in:   "failed to connect to `user=postgres database=replica_dwh`:\n\t" + seg + "\n\t" + seg,
			want: "failed to connect to `user=postgres database=replica_dwh`: " + seg,
		},
		{
			name: "single attempt is left intact",
			in:   "failed to connect to `user=postgres database=replica_dwh`: " + seg,
			want: "failed to connect to `user=postgres database=replica_dwh`: " + seg,
		},
		{
			name: "two DIFFERENT attempts are not collapsed",
			in:   "failed to connect to `db=x`: " + seg + " other error here",
			want: "failed to connect to `db=x`: " + seg + " other error here",
		},
		{
			name: "message without the conninfo marker is only whitespace-normalized",
			in:   "some\n\tother   error",
			want: "some other error",
		},
	}
	for _, tc := range tests {
		if got := cleanConnectError(tc.in); got != tc.want {
			t.Errorf("%s:\n got:  %q\n want: %q", tc.name, got, tc.want)
		}
	}
}
