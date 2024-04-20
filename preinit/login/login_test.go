package login

import (
	"testing"

	"github.com/cloudboss/easyto/preinit/constants"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestAddSystemUser(t *testing.T) {
	setup := func(passwd, shadow, group, gshadow *string) afero.Fs {
		fs := afero.NewMemMapFs()
		if passwd != nil {
			afero.WriteFile(fs, constants.FileEtcPasswd, []byte(*passwd), constants.ModeEtcPasswd)
		}
		if shadow != nil {
			afero.WriteFile(fs, constants.FileEtcShadow, []byte(*shadow), constants.ModeEtcShadow)
		}
		if group != nil {
			afero.WriteFile(fs, constants.FileEtcGroup, []byte(*group), constants.ModeEtcGroup)
		}
		if gshadow != nil {
			afero.WriteFile(fs, constants.FileEtcGShadow, []byte(*gshadow), constants.ModeEtcGShadow)
		}
		return fs
	}
	read := func(fs afero.Fs, path string) *string {
		b, _ := afero.ReadFile(fs, path)
		s := string(b)
		return &s
	}
	exists := func(fs afero.Fs, path string) bool {
		x, _ := afero.Exists(fs, path)
		return x
	}
	testCases := []struct {
		description   string
		passwd        *string
		passwdResult  *string
		shadow        *string
		shadowResult  *string
		group         *string
		groupResult   *string
		gshadow       *string
		gshadowResult *string
		username      string
		groupname     string
		err           error
	}{
		{
			description:   "Invalid username",
			passwd:        nil,
			passwdResult:  nil,
			shadow:        nil,
			shadowResult:  nil,
			group:         nil,
			groupResult:   nil,
			gshadow:       nil,
			gshadowResult: nil,
			username:      "",
			groupname:     "",
			err:           ErrUsernameLength,
		},
		{
			description:   "Invalid groupname",
			passwd:        nil,
			passwdResult:  nil,
			shadow:        nil,
			shadowResult:  nil,
			group:         nil,
			groupResult:   nil,
			gshadow:       nil,
			gshadowResult: nil,
			username:      "xyz",
			groupname:     "",
			err:           ErrGroupnameLength,
		},
		{
			description:   "No files exist",
			passwd:        nil,
			passwdResult:  p("xyz:x:100:100:xyz:/nonexistent:/bin/false\n"),
			shadow:        nil,
			shadowResult:  p("xyz:!!:0:0:99999:7:::\n"),
			group:         nil,
			groupResult:   p("xyz:x:100:xyz\n"),
			gshadow:       nil,
			gshadowResult: p("xyz:!!::xyz\n"),
			username:      "xyz",
			groupname:     "xyz",
			err:           nil,
		},
		{
			description: "Passwd and group files exist",
			passwd:      p("abc:x:100:100:abc:/nonexistent:/bin/false\n"),
			passwdResult: p(`abc:x:100:100:abc:/nonexistent:/bin/false
xyz:x:101:101:xyz:/nonexistent:/bin/false
`),
			shadow:        nil,
			shadowResult:  nil,
			group:         p("abc:x:100:\n"),
			groupResult:   p("abc:x:100:\nxyz:x:101:xyz\n"),
			gshadow:       nil,
			gshadowResult: nil,
			username:      "xyz",
			groupname:     "xyz",
			err:           nil,
		},
		{
			description: "Passwd, group, and shadow files exist",
			passwd:      p("abc:x:100:100:abc:/nonexistent:/bin/false\n"),
			passwdResult: p(`abc:x:100:100:abc:/nonexistent:/bin/false
xyz:x:101:101:xyz:/nonexistent:/bin/false
`),
			shadow: p("abc:!!:0:0:99999:7:::\n"),
			shadowResult: p(`abc:!!:0:0:99999:7:::
xyz:!!:0:0:99999:7:::
`),
			group:         p("abc:x:100:\n"),
			groupResult:   p("abc:x:100:\nxyz:x:101:xyz\n"),
			gshadow:       nil,
			gshadowResult: nil,
			username:      "xyz",
			groupname:     "xyz",
			err:           nil,
		},
		{
			description: "Passwd, group, and gshadow files exist",
			passwd:      p("abc:x:100:100:abc:/nonexistent:/bin/false\n"),
			passwdResult: p(`abc:x:100:100:abc:/nonexistent:/bin/false
xyz:x:101:101:xyz:/nonexistent:/bin/false
`),
			shadow:        nil,
			shadowResult:  nil,
			group:         p("abc:x:100:\n"),
			groupResult:   p("abc:x:100:\nxyz:x:101:xyz\n"),
			gshadow:       p("abc:!!::\n"),
			gshadowResult: p("abc:!!::\nxyz:!!::xyz\n"),
			username:      "xyz",
			groupname:     "xyz",
			err:           nil,
		},
		{
			description: "All files exist",
			passwd:      p("rpc:x:32:32:Rpcbind Daemon:/var/lib/rpcbind:/sbin/nologin\n"),
			passwdResult: p(`rpc:x:32:32:Rpcbind Daemon:/var/lib/rpcbind:/sbin/nologin
xyz:x:100:100:xyz:/nonexistent:/bin/false
`),
			shadow: p("rpc:!!:19460:0:99999:7:::\n"),
			shadowResult: p(`rpc:!!:19460:0:99999:7:::
xyz:!!:0:0:99999:7:::
`),
			group:         p("rpc:x:32:\n"),
			groupResult:   p("rpc:x:32:\nxyz:x:100:xyz\n"),
			gshadow:       p("abc:!!::\n"),
			gshadowResult: p("abc:!!::\nxyz:!!::xyz\n"),
			username:      "xyz",
			groupname:     "xyz",
			err:           nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fs := setup(tc.passwd, tc.shadow, tc.group, tc.gshadow)
			_, _, err := AddSystemUser(fs, tc.username, tc.groupname, "/nonexistent")
			assert.Equal(t, tc.err, err)
			if err == nil {
				if tc.passwdResult != nil {
					assert.Equal(t, *tc.passwdResult, *read(fs, constants.FileEtcPasswd))
				} else {
					assert.False(t, exists(fs, constants.FileEtcPasswd))
				}
				if tc.shadowResult != nil {
					assert.Equal(t, *tc.shadowResult, *read(fs, constants.FileEtcShadow))
				} else {
					assert.False(t, exists(fs, constants.FileEtcShadow))
				}
				if tc.groupResult != nil {
					assert.Equal(t, *tc.groupResult, *read(fs, constants.FileEtcGroup))
				} else {
					assert.False(t, exists(fs, constants.FileEtcGroup))
				}
				if tc.gshadowResult != nil {
					assert.Equal(t, *tc.gshadowResult, *read(fs, constants.FileEtcGShadow))
				} else {
					assert.False(t, exists(fs, constants.FileEtcGShadow))
				}
			}
		})
	}
}

func p[T any](v T) *T {
	return &v
}
