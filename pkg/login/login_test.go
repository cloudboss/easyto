package login

import (
	"path/filepath"
	"testing"

	"github.com/cloudboss/easyto/pkg/constants"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func loginSetup(passwd, shadow, group, gshadow *string, baseDir string) afero.Fs {
	fs := afero.NewMemMapFs()
	if passwd != nil {
		fileEtcPasswd := filepath.Join(baseDir, constants.FileEtcPasswd)
		afero.WriteFile(fs, fileEtcPasswd, []byte(*passwd), constants.ModeEtcPasswd)
	}
	if shadow != nil {
		fileEtcShadow := filepath.Join(baseDir, constants.FileEtcShadow)
		afero.WriteFile(fs, fileEtcShadow, []byte(*shadow), constants.ModeEtcShadow)
	}
	if group != nil {
		fileEtcGroup := filepath.Join(baseDir, constants.FileEtcGroup)
		afero.WriteFile(fs, fileEtcGroup, []byte(*group), constants.ModeEtcGroup)
	}
	if gshadow != nil {
		fileEtcGShadow := filepath.Join(baseDir, constants.FileEtcGShadow)
		afero.WriteFile(fs, fileEtcGShadow, []byte(*gshadow), constants.ModeEtcGShadow)
	}
	return fs
}

func loginRead(fs afero.Fs, path string) *string {
	b, _ := afero.ReadFile(fs, path)
	s := string(b)
	return &s
}

func loginExists(fs afero.Fs, path string) bool {
	x, _ := afero.Exists(fs, path)
	return x
}

func TestAddSystemUser(t *testing.T) {
	testCases := []struct {
		baseDir       string
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
			description: "Invalid username",
			username:    "",
			groupname:   "",
			err:         ErrUsernameLength,
		},
		{
			description: "Invalid groupname",
			username:    "xyz",
			groupname:   "",
			err:         ErrGroupnameLength,
		},
		{
			description:   "No files exist",
			passwdResult:  p("xyz:x:100:100:xyz:/nonexistent:/bin/false\n"),
			shadowResult:  p("xyz:!!:0:0:99999:7:::\n"),
			groupResult:   p("xyz:x:100:xyz\n"),
			gshadowResult: p("xyz:!!::xyz\n"),
			username:      "xyz",
			groupname:     "xyz",
		},
		{
			description: "Passwd and group files exist",
			passwd:      p("abc:x:100:100:abc:/nonexistent:/bin/false\n"),
			passwdResult: p(`abc:x:100:100:abc:/nonexistent:/bin/false
xyz:x:101:101:xyz:/nonexistent:/bin/false
`),
			group:       p("abc:x:100:\n"),
			groupResult: p("abc:x:100:\nxyz:x:101:xyz\n"),
			username:    "xyz",
			groupname:   "xyz",
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
			group:       p("abc:x:100:\n"),
			groupResult: p("abc:x:100:\nxyz:x:101:xyz\n"),
			username:    "xyz",
			groupname:   "xyz",
		},
		{
			description: "Passwd, group, and gshadow files exist",
			passwd:      p("abc:x:100:100:abc:/nonexistent:/bin/false\n"),
			passwdResult: p(`abc:x:100:100:abc:/nonexistent:/bin/false
xyz:x:101:101:xyz:/nonexistent:/bin/false
`),
			group:         p("abc:x:100:\n"),
			groupResult:   p("abc:x:100:\nxyz:x:101:xyz\n"),
			gshadow:       p("abc:!!::\n"),
			gshadowResult: p("abc:!!::\nxyz:!!::xyz\n"),
			username:      "xyz",
			groupname:     "xyz",
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
		},
		{
			baseDir:       "/base",
			description:   "No files exist with basedir",
			passwdResult:  p("xyz:x:100:100:xyz:/nonexistent:/bin/false\n"),
			shadowResult:  p("xyz:!!:0:0:99999:7:::\n"),
			groupResult:   p("xyz:x:100:xyz\n"),
			gshadowResult: p("xyz:!!::xyz\n"),
			username:      "xyz",
			groupname:     "xyz",
		},
		{
			description:   "User and group exist",
			passwd:        p("xyz:x:100:100:xyz:/nonexistent:/bin/false\n"),
			passwdResult:  p("xyz:x:100:100:xyz:/nonexistent:/bin/false\n"),
			shadow:        p("xyz:!!:0:0:99999:7:::\n"),
			shadowResult:  p("xyz:!!:0:0:99999:7:::\n"),
			group:         p("xyz:x:100:xyz\n"),
			groupResult:   p("xyz:x:100:xyz\n"),
			gshadow:       p("xyz:!!::xyz\n"),
			gshadowResult: p("xyz:!!::xyz\n"),
			username:      "xyz",
			groupname:     "xyz",
			err:           ErrUsernameExists,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fs := loginSetup(tc.passwd, tc.shadow, tc.group, tc.gshadow, tc.baseDir)
			_, _, err := AddSystemUser(fs, tc.username, tc.groupname, "/nonexistent", tc.baseDir)
			assert.Equal(t, tc.err, err)
			if err == nil {
				fileEtcPasswd := filepath.Join(tc.baseDir, constants.FileEtcPasswd)
				fileEtcShadow := filepath.Join(tc.baseDir, constants.FileEtcShadow)
				fileEtcGroup := filepath.Join(tc.baseDir, constants.FileEtcGroup)
				fileEtcGShadow := filepath.Join(tc.baseDir, constants.FileEtcGShadow)
				if tc.passwdResult != nil {
					assert.Equal(t, *tc.passwdResult, *loginRead(fs, fileEtcPasswd))
				} else {
					assert.False(t, loginExists(fs, fileEtcPasswd))
				}
				if tc.shadowResult != nil {
					assert.Equal(t, *tc.shadowResult, *loginRead(fs, fileEtcShadow))
				} else {
					assert.False(t, loginExists(fs, constants.FileEtcShadow))
				}
				if tc.groupResult != nil {
					assert.Equal(t, *tc.groupResult, *loginRead(fs, fileEtcGroup))
				} else {
					assert.False(t, loginExists(fs, constants.FileEtcGroup))
				}
				if tc.gshadowResult != nil {
					assert.Equal(t, *tc.gshadowResult, *loginRead(fs, fileEtcGShadow))
				} else {
					assert.False(t, loginExists(fs, fileEtcGShadow))
				}
			}
		})
	}
}

func TestAddLoginUser(t *testing.T) {
	testCases := []struct {
		baseDir       string
		homeDir       string
		shell         string
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
			description: "Invalid username",
			err:         ErrUsernameLength,
		},
		{
			description: "Invalid groupname",
			username:    "xyz",
			err:         ErrGroupnameLength,
		},
		{
			description:   "No files exist",
			homeDir:       "/home/xyz",
			shell:         "/bin/sh",
			passwdResult:  p("xyz:x:1000:1000:xyz:/home/xyz:/bin/sh\n"),
			shadowResult:  p("xyz:*::0:99999:7:::\n"),
			groupResult:   p("wheel:x:10:xyz\nxyz:x:1000:xyz\n"),
			gshadowResult: p("wheel:::xyz\nxyz:!!::xyz\n"),
			username:      "xyz",
			groupname:     "xyz",
		},
		{
			description: "Passwd and group files exist",
			homeDir:     "/home/xyz",
			shell:       "/bin/sh",
			passwd:      p("abc:x:1000:1000:abc:/home/abc:/bin/bash\n"),
			passwdResult: p(`abc:x:1000:1000:abc:/home/abc:/bin/bash
xyz:x:1001:1001:xyz:/home/xyz:/bin/sh
`),
			group:       p("abc:x:1000:\n"),
			groupResult: p("abc:x:1000:\nwheel:x:10:xyz\nxyz:x:1001:xyz\n"),
			username:    "xyz",
			groupname:   "xyz",
		},
		{
			description: "Passwd, group, and shadow files exist",
			homeDir:     "/home/xyz",
			shell:       "/bin/sh",
			passwd:      p("abc:x:1000:1000:abc:/home/abc:/bin/sh\n"),
			passwdResult: p(`abc:x:1000:1000:abc:/home/abc:/bin/sh
xyz:x:1001:1001:xyz:/home/xyz:/bin/sh
`),
			shadow: p("abc:*:0:0:99999:7:::\n"),
			shadowResult: p(`abc:*:0:0:99999:7:::
xyz:*::0:99999:7:::
`),
			group:       p("abc:x:1000:\n"),
			groupResult: p("abc:x:1000:\nwheel:x:10:xyz\nxyz:x:1001:xyz\n"),
			username:    "xyz",
			groupname:   "xyz",
		},
		{
			description: "Passwd, group, and gshadow files exist",
			homeDir:     "/home/xyz",
			shell:       "/bin/sh",
			passwd:      p("abc:x:1000:1000:abc:/home/abc:/bin/bash\n"),
			passwdResult: p(`abc:x:1000:1000:abc:/home/abc:/bin/bash
xyz:x:1001:1001:xyz:/home/xyz:/bin/sh
`),
			group:         p("abc:x:1000:\n"),
			groupResult:   p("abc:x:1000:\nwheel:x:10:xyz\nxyz:x:1001:xyz\n"),
			gshadow:       p("abc:!!::\n"),
			gshadowResult: p("abc:!!::\nwheel:::xyz\nxyz:!!::xyz\n"),
			username:      "xyz",
			groupname:     "xyz",
		},
		{
			description: "All files exist",
			homeDir:     "/home/xyz",
			shell:       "/bin/bash",
			passwd:      p("rpc:x:32:32:Rpcbind Daemon:/var/lib/rpcbind:/sbin/nologin\n"),
			passwdResult: p(`rpc:x:32:32:Rpcbind Daemon:/var/lib/rpcbind:/sbin/nologin
xyz:x:1000:1000:xyz:/home/xyz:/bin/bash
`),
			shadow: p("rpc:!!:19460:0:99999:7:::\n"),
			shadowResult: p(`rpc:!!:19460:0:99999:7:::
xyz:*::0:99999:7:::
`),
			group:         p("rpc:x:32:\n"),
			groupResult:   p("rpc:x:32:\nwheel:x:10:xyz\nxyz:x:1000:xyz\n"),
			gshadow:       p("abc:!!::\n"),
			gshadowResult: p("abc:!!::\nwheel:::xyz\nxyz:!!::xyz\n"),
			username:      "xyz",
			groupname:     "xyz",
		},
		{
			description:   "No files exist with basedir",
			baseDir:       "/base",
			homeDir:       "/home/xyz",
			shell:         "/bin/bash",
			passwdResult:  p("xyz:x:1000:1000:xyz:/home/xyz:/bin/bash\n"),
			shadowResult:  p("xyz:*::0:99999:7:::\n"),
			groupResult:   p("wheel:x:10:xyz\nxyz:x:1000:xyz\n"),
			gshadowResult: p("wheel:::xyz\nxyz:!!::xyz\n"),
			username:      "xyz",
			groupname:     "xyz",
		},
		{
			description: "Wheel group exists",
			homeDir:     "/home/xyz",
			shell:       "/bin/bash",
			passwd:      p("rpc:x:32:32:Rpcbind Daemon:/var/lib/rpcbind:/sbin/nologin\n"),
			passwdResult: p(`rpc:x:32:32:Rpcbind Daemon:/var/lib/rpcbind:/sbin/nologin
xyz:x:1000:1000:xyz:/home/xyz:/bin/bash
`),
			shadow: p("rpc:!!:19460:0:99999:7:::\n"),
			shadowResult: p(`rpc:!!:19460:0:99999:7:::
xyz:*::0:99999:7:::
`),
			group:         p("wheel:x:10:\nrpc:x:32:\n"),
			groupResult:   p("wheel:x:10:xyz\nrpc:x:32:\nxyz:x:1000:xyz\n"),
			gshadow:       p("wheel:::\nrpc:!::\nabc:!!::\n"),
			gshadowResult: p("wheel:::xyz\nrpc:!::\nabc:!!::\nxyz:!!::xyz\n"),
			username:      "xyz",
			groupname:     "xyz",
		},
		{
			description: "User and group exists",
			homeDir:     "/home/xyz",
			shell:       "/bin/bash",
			passwd: p(`rpc:x:32:32:Rpcbind Daemon:/var/lib/rpcbind:/sbin/nologin
xyz:x:1000:1000:xyz:/home/xyz:/bin/bash
`),
			passwdResult: p(`rpc:x:32:32:Rpcbind Daemon:/var/lib/rpcbind:/sbin/nologin
xyz:x:1000:1000:xyz:/home/xyz:/bin/bash
`),
			shadow: p(`rpc:!!:19460:0:99999:7:::
xyz:*:0:0:99999:7:::
`),
			shadowResult: p(`rpc:!!:19460:0:99999:7:::
xyz:*:0:0:99999:7:::
`),
			group:         p("wheel:x:10:xyz\nrpc:x:32:\nxyz:x:1000:xyz\n"),
			groupResult:   p("wheel:x:10:xyz\nrpc:x:32:\nxyz:x:1000:xyz\n"),
			gshadow:       p("wheel:::xyz\nrpc:!::\nabc:!!::\nxyz:!!::xyz\n"),
			gshadowResult: p("wheel:::xyz\nrpc:!::\nabc:!!::\nxyz:!!::xyz\n"),
			username:      "xyz",
			groupname:     "xyz",
			err:           ErrUsernameExists,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fs := loginSetup(tc.passwd, tc.shadow, tc.group, tc.gshadow, tc.baseDir)
			_, _, err := AddLoginUser(fs, tc.username, tc.groupname, tc.homeDir, tc.shell, tc.baseDir)
			assert.Equal(t, tc.err, err)
			if err == nil {
				fileEtcPasswd := filepath.Join(tc.baseDir, constants.FileEtcPasswd)
				fileEtcShadow := filepath.Join(tc.baseDir, constants.FileEtcShadow)
				fileEtcGroup := filepath.Join(tc.baseDir, constants.FileEtcGroup)
				fileEtcGShadow := filepath.Join(tc.baseDir, constants.FileEtcGShadow)
				if tc.passwdResult != nil {
					assert.Equal(t, *tc.passwdResult, *loginRead(fs, fileEtcPasswd))
				} else {
					assert.False(t, loginExists(fs, fileEtcPasswd))
				}
				if tc.shadowResult != nil {
					assert.Equal(t, *tc.shadowResult, *loginRead(fs, fileEtcShadow))
				} else {
					assert.False(t, loginExists(fs, constants.FileEtcShadow))
				}
				if tc.groupResult != nil {
					assert.Equal(t, *tc.groupResult, *loginRead(fs, fileEtcGroup))
				} else {
					assert.False(t, loginExists(fs, constants.FileEtcGroup))
				}
				if tc.gshadowResult != nil {
					assert.Equal(t, *tc.gshadowResult, *loginRead(fs, fileEtcGShadow))
				} else {
					assert.Equal(t, "", *loginRead(fs, fileEtcGShadow))
					assert.False(t, loginExists(fs, fileEtcGShadow))
				}
			}
		})
	}
}

func TestAddRootUser(t *testing.T) {
	testCases := []struct {
		baseDir       string
		shell         string
		description   string
		passwd        *string
		passwdResult  *string
		shadow        *string
		shadowResult  *string
		group         *string
		groupResult   *string
		gshadow       *string
		gshadowResult *string
		err           error
	}{
		{
			description:  "No root user exists",
			shell:        "/bin/sh",
			passwd:       p("bin:x:1:1:bin:/bin:/sbin/nologin\n"),
			passwdResult: p("bin:x:1:1:bin:/bin:/sbin/nologin\nroot:x:0:0:root:/root:/bin/sh\n"),
			shadow:       p("bin:!::0:::::\n"),
			shadowResult: p("bin:!::0:::::\nroot:*:0:0:99999:7:::\n"),
			group:        p("bin:x:1:root,bin,daemon\n"),
			groupResult:  p("bin:x:1:root,bin,daemon\nroot:x:0:root\n"),
		},
		{
			description:  "Root user exists",
			shell:        "/bin/abc", // Shell not written because user exists.
			passwd:       p("bin:x:1:1:bin:/bin:/sbin/nologin\nroot:x:0:0:root:/root:/bin/sh\n"),
			passwdResult: p("bin:x:1:1:bin:/bin:/sbin/nologin\nroot:x:0:0:root:/root:/bin/sh\n"),
			shadow:       p("bin:!::0:::::\nroot:*:0:0:99999:7:::\n"),
			shadowResult: p("bin:!::0:::::\nroot:*:0:0:99999:7:::\n"),
			group:        p("bin:x:1:root,bin,daemon\nroot:x:0:root\n"),
			groupResult:  p("bin:x:1:root,bin,daemon\nroot:x:0:root\n"),
			err:          ErrUsernameExists,
		},
		{
			description:  "UID 0 exists under another username",
			shell:        "/bin/sh",
			passwd:       p("bin:x:1:1:bin:/bin:/sbin/nologin\ntoor:x:0:0:toor:/toor:/bin/sh\n"),
			passwdResult: p("bin:x:1:1:bin:/bin:/sbin/nologin\ntoor:x:0:0:toor:/toor:/bin/sh\n"),
			shadow:       p("bin:!::0:::::\ntoor:*:0:0:99999:7:::\n"),
			shadowResult: p("bin:!::0:::::\ntoor:*:0:0:99999:7:::\n"),
			group:        p("bin:x:1:root,bin,daemon\ntoor:x:0:toor\n"),
			groupResult:  p("bin:x:1:root,bin,daemon\ntoor:x:0:toor\n"),
			err:          ErrNoAvailableIDs,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fs := loginSetup(tc.passwd, tc.shadow, tc.group, tc.gshadow, tc.baseDir)
			_, _, err := AddRootUser(fs, tc.shell, tc.baseDir)
			assert.Equal(t, tc.err, err)
			if err == nil {
				fileEtcPasswd := filepath.Join(tc.baseDir, constants.FileEtcPasswd)
				fileEtcShadow := filepath.Join(tc.baseDir, constants.FileEtcShadow)
				fileEtcGroup := filepath.Join(tc.baseDir, constants.FileEtcGroup)
				fileEtcGShadow := filepath.Join(tc.baseDir, constants.FileEtcGShadow)
				if tc.passwdResult != nil {
					assert.Equal(t, *tc.passwdResult, *loginRead(fs, fileEtcPasswd))
				} else {
					assert.False(t, loginExists(fs, fileEtcPasswd))
				}
				if tc.shadowResult != nil {
					assert.Equal(t, *tc.shadowResult, *loginRead(fs, fileEtcShadow))
				} else {
					assert.False(t, loginExists(fs, constants.FileEtcShadow))
				}
				if tc.groupResult != nil {
					assert.Equal(t, *tc.groupResult, *loginRead(fs, fileEtcGroup))
				} else {
					assert.False(t, loginExists(fs, constants.FileEtcGroup))
				}
				if tc.gshadowResult != nil {
					assert.Equal(t, *tc.gshadowResult, *loginRead(fs, fileEtcGShadow))
				} else {
					assert.Equal(t, "", *loginRead(fs, fileEtcGShadow))
					assert.False(t, loginExists(fs, fileEtcGShadow))
				}
			}
		})
	}
}

func p[T any](v T) *T {
	return &v
}
