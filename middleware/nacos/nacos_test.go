package nacos

import "testing"

func TestDesensitizeConfigContent(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "pwd lowercase", in: "pwd: abc", want: "pwd: ***"},
		{name: "pwd with prefix", in: "default-pwd: abc", want: "default-pwd: ***"},
		{name: "pwd mixed case with suffix", in: "legacyP12Pwd: abc", want: "legacyP12Pwd: ***"},
		{name: "passwd", in: "passwd: abc", want: "passwd: ***"},
		{name: "passwd mixed case with suffix", in: "appPasswd: abc", want: "appPasswd: ***"},
		{name: "passwd with suffix", in: "passwdHash: abc", want: "passwdHash: ***"},
		{name: "credential", in: "credential: abc", want: "credential: ***"},
		{name: "credential with prefix and suffix", in: "clientCredentialList: abc", want: "clientCredentialList: ***"},
		{name: "authorization lowercase", in: "authorization: Bearer abc", want: "authorization: ***"},
		{name: "authorization mixed case", in: "Authorization: Bearer abc", want: "Authorization: ***"},
		{name: "authorization with suffix", in: "authorizationHeader: abc", want: "authorizationHeader: ***"},
		{name: "pem", in: "pem: abc", want: "pem: ***"},
		{name: "pem with suffix", in: "pemContent: abc", want: "pemContent: ***"},
		{name: "pem inside key", in: "signingPemBlock: abc", want: "signingPemBlock: ***"},
		{name: "pem suffix short", in: "certPem: abc", want: "certPem: ***"},
		{name: "pem uppercase suffix", in: "privateKeyPEM: abc", want: "privateKeyPEM: ***"},
		{name: "username is untouched", in: "username: abc", want: "username: abc"},
		{name: "non-secret key with a keyword inside loses its value", in: "hotkey: 1", want: "hotkey: ***"},
		{name: "existing secret-key still masked", in: "secret-key: abc", want: "secret-key: ***"},
		{name: "existing access-token still masked", in: "access-token: abc", want: "access-token: ***"},
		{name: "existing password still masked", in: "password: abc", want: "password: ***"},
		{name: "multi-line keeps untouched lines", in: "username: bob\npassword: x\n", want: "username: bob\npassword: ***\n"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := desensitizeConfigContent(c.in); got != c.want {
				t.Errorf("desensitizeConfigContent(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
