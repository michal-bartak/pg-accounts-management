package pg

import "testing"

func TestCleanConnectError(t *testing.T) {
	seg := "127.0.0.1:5002 (127.0.0.1): dial error: dial tcp 127.0.0.1:5002: connect: connection refused"
	seg6 := "db.example:5002 (::1): dial error: dial tcp [::1]:5002: connect: connection refused"
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
			// The real shape: pgx dials every address with TLS, then every address plaintext, so
			// the repeats are interleaved ("A B A B") — the old exact-halves check left it doubled.
			// This is what `localhost` on a dead port actually returns.
			name: "interleaved TLS+plaintext attempts per address collapse to one each",
			in:   "failed to connect to `db=x`: " + seg6 + " " + seg + " " + seg6 + " " + seg,
			want: "failed to connect to `db=x`: " + seg6 + " " + seg,
		},
		{
			name: "adjacent duplicates per address also collapse",
			in:   "failed to connect to `db=x`: " + seg6 + " " + seg6 + " " + seg + " " + seg,
			want: "failed to connect to `db=x`: " + seg6 + " " + seg,
		},
		{
			name: "three identical attempts collapse to one",
			in:   "failed to connect to `db=x`: " + seg + " " + seg + " " + seg,
			want: "failed to connect to `db=x`: " + seg,
		},
		{
			name: "two genuinely different addresses are both kept",
			in:   "failed to connect to `db=x`: " + seg6 + " " + seg,
			want: "failed to connect to `db=x`: " + seg6 + " " + seg,
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
