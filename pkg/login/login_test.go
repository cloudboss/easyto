package login

import (
	"os"
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

// Test parsing functions

func TestParsePasswd(t *testing.T) {
	testCases := []struct {
		description string
		content     string
		wantErr     bool
		wantUIDs    []uint16
		wantNames   []string
	}{
		{
			description: "Valid passwd file",
			content: `root:x:0:0:root:/root:/bin/bash
bin:x:1:1:bin:/bin:/sbin/nologin
daemon:x:2:2:daemon:/sbin:/sbin/nologin
`,
			wantErr:   false,
			wantUIDs:  []uint16{0, 1, 2},
			wantNames: []string{"root", "bin", "daemon"},
		},
		{
			description: "Invalid field count",
			content:     "root:x:0:0:root:/root\n",
			wantErr:     true,
		},
		{
			description: "Invalid UID",
			content:     "root:x:invalid:0:root:/root:/bin/bash\n",
			wantErr:     true,
		},
		{
			description: "Invalid GID",
			content:     "root:x:0:invalid:root:/root:/bin/bash\n",
			wantErr:     true,
		},
		{
			description: "Empty file",
			content:     "",
			wantErr:     false,
			wantUIDs:    []uint16{},
			wantNames:   []string{},
		},
		{
			description: "UID overflow",
			content:     "test:x:70000:0:test:/:/bin/sh\n",
			wantErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			passwdFile := "/etc/passwd"
			afero.WriteFile(fs, passwdFile, []byte(tc.content), 0644)

			byUID, byName, list, err := ParsePasswd(fs, passwdFile)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tc.wantUIDs), len(byUID))
				assert.Equal(t, len(tc.wantNames), len(byName))
				assert.Equal(t, len(tc.wantUIDs), len(list))

				for _, uid := range tc.wantUIDs {
					assert.Contains(t, byUID, uid)
				}
				for _, name := range tc.wantNames {
					assert.Contains(t, byName, name)
				}
			}
		})
	}
}

func TestParsePasswdFileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	_, _, _, err := ParsePasswd(fs, "/nonexistent")
	assert.Error(t, err)
}

func TestParseGroup(t *testing.T) {
	testCases := []struct {
		description string
		content     string
		wantErr     bool
		wantGIDs    []uint16
		wantNames   []string
	}{
		{
			description: "Valid group file",
			content: `root:x:0:
bin:x:1:root,bin,daemon
wheel:x:10:alice,bob
`,
			wantErr:   false,
			wantGIDs:  []uint16{0, 1, 10},
			wantNames: []string{"root", "bin", "wheel"},
		},
		{
			description: "Invalid field count",
			content:     "root:x:0\n",
			wantErr:     true,
		},
		{
			description: "Invalid GID",
			content:     "root:x:invalid:\n",
			wantErr:     true,
		},
		{
			description: "Empty users list",
			content:     "root:x:0:\n",
			wantErr:     false,
			wantGIDs:    []uint16{0},
			wantNames:   []string{"root"},
		},
		{
			description: "Multiple users",
			content:     "wheel:x:10:alice,bob,charlie\n",
			wantErr:     false,
			wantGIDs:    []uint16{10},
			wantNames:   []string{"wheel"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			groupFile := "/etc/group"
			afero.WriteFile(fs, groupFile, []byte(tc.content), 0644)

			byGID, byName, list, err := ParseGroup(fs, groupFile)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tc.wantGIDs), len(byGID))
				assert.Equal(t, len(tc.wantNames), len(byName))
				assert.Equal(t, len(tc.wantGIDs), len(list))

				for _, gid := range tc.wantGIDs {
					assert.Contains(t, byGID, gid)
				}
				for _, name := range tc.wantNames {
					assert.Contains(t, byName, name)
				}
			}
		})
	}
}

func TestParseGroupFileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	_, _, _, err := ParseGroup(fs, "/nonexistent")
	assert.Error(t, err)
}

func TestParseShadow(t *testing.T) {
	testCases := []struct {
		description string
		content     string
		wantErr     bool
		wantNames   []string
	}{
		{
			description: "Valid shadow file",
			content: `root:*:19460:0:99999:7:::
bin:!!:19460::::::
daemon:!!:19460:0:99999:7:10:20:30
`,
			wantErr:   false,
			wantNames: []string{"root", "bin", "daemon"},
		},
		{
			description: "Invalid field count",
			content:     "root:*:19460:0:99999:7::\n",
			wantErr:     true,
		},
		{
			description: "Invalid lastChange",
			content:     "root:*:invalid:0:99999:7:::\n",
			wantErr:     true,
		},
		{
			description: "Empty optional fields",
			content:     "root:*:::::::\n",
			wantErr:     false,
			wantNames:   []string{"root"},
		},
		{
			description: "Invalid minAge",
			content:     "root:*:19460:invalid:99999:7:::\n",
			wantErr:     true,
		},
		{
			description: "Invalid maxAge",
			content:     "root:*:19460:0:invalid:7:::\n",
			wantErr:     true,
		},
		{
			description: "Invalid warningPeriod",
			content:     "root:*:19460:0:99999:invalid:::\n",
			wantErr:     true,
		},
		{
			description: "Invalid inactivityPeriod",
			content:     "root:*:19460:0:99999:7:invalid::\n",
			wantErr:     true,
		},
		{
			description: "Invalid expiration",
			content:     "root:*:19460:0:99999:7::invalid:\n",
			wantErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			shadowFile := "/etc/shadow"
			afero.WriteFile(fs, shadowFile, []byte(tc.content), 0600)

			byName, list, err := ParseShadow(fs, shadowFile)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tc.wantNames), len(byName))
				assert.Equal(t, len(tc.wantNames), len(list))

				for _, name := range tc.wantNames {
					assert.Contains(t, byName, name)
				}
			}
		})
	}
}

func TestParseShadowFileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	_, _, err := ParseShadow(fs, "/nonexistent")
	assert.Error(t, err)
}

func TestParseGShadow(t *testing.T) {
	testCases := []struct {
		description string
		content     string
		wantErr     bool
		wantNames   []string
	}{
		{
			description: "Valid gshadow file",
			content: `root:::
wheel:::alice,bob
bin:!!:admin1,admin2:user1,user2
`,
			wantErr:   false,
			wantNames: []string{"root", "wheel", "bin"},
		},
		{
			description: "Invalid field count",
			content:     "root::\n",
			wantErr:     true,
		},
		{
			description: "Empty admins and users",
			content:     "root:::\n",
			wantErr:     false,
			wantNames:   []string{"root"},
		},
		{
			description: "Only admins",
			content:     "wheel:!!:admin1:\n",
			wantErr:     false,
			wantNames:   []string{"wheel"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			gshadowFile := "/etc/gshadow"
			afero.WriteFile(fs, gshadowFile, []byte(tc.content), 0600)

			byName, list, err := ParseGShadow(fs, gshadowFile)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tc.wantNames), len(byName))
				assert.Equal(t, len(tc.wantNames), len(list))

				for _, name := range tc.wantNames {
					assert.Contains(t, byName, name)
				}
			}
		})
	}
}

func TestParseGShadowFileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	_, _, err := ParseGShadow(fs, "/nonexistent")
	assert.Error(t, err)
}

// Test String() methods

func TestPasswdEntryString(t *testing.T) {
	entry := PasswdEntry{
		Username: "testuser",
		Password: "x",
		UID:      1000,
		GID:      1000,
		Comment:  "Test User",
		HomeDir:  "/home/testuser",
		Shell:    "/bin/bash",
	}
	expected := "testuser:x:1000:1000:Test User:/home/testuser:/bin/bash"
	assert.Equal(t, expected, entry.String())
}

func TestGroupEntryString(t *testing.T) {
	testCases := []struct {
		description string
		entry       GroupEntry
		expected    string
	}{
		{
			description: "With users",
			entry: GroupEntry{
				Groupname: "testgroup",
				Password:  "x",
				GID:       1000,
				Users:     []string{"alice", "bob"},
			},
			expected: "testgroup:x:1000:alice,bob",
		},
		{
			description: "No users",
			entry: GroupEntry{
				Groupname: "testgroup",
				Password:  "x",
				GID:       1000,
				Users:     []string{},
			},
			expected: "testgroup:x:1000:",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.entry.String())
		})
	}
}

func TestShadowEntryString(t *testing.T) {
	testCases := []struct {
		description string
		entry       ShadowEntry
		expected    string
	}{
		{
			description: "All fields populated",
			entry: ShadowEntry{
				Username:         "testuser",
				Password:         "*",
				LastChange:       19460,
				MinAge:           0,
				MaxAge:           99999,
				WarningPeriod:    7,
				InactivityPeriod: 10,
				Expiration:       20,
				Unused:           "",
			},
			expected: "testuser:*:19460:0:99999:7:10:20:",
		},
		{
			description: "Empty optional fields (negative values)",
			entry: ShadowEntry{
				Username:         "testuser",
				Password:         "!!",
				LastChange:       -1,
				MinAge:           -1,
				MaxAge:           -1,
				WarningPeriod:    -1,
				InactivityPeriod: -1,
				Expiration:       -1,
				Unused:           "",
			},
			expected: "testuser:!!:::::::",
		},
		{
			description: "With unused field",
			entry: ShadowEntry{
				Username:         "testuser",
				Password:         "!!",
				LastChange:       -1,
				MinAge:           -1,
				MaxAge:           -1,
				WarningPeriod:    -1,
				InactivityPeriod: -1,
				Expiration:       -1,
				Unused:           "reserved",
			},
			expected: "testuser:!!:::::::reserved",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.entry.String())
		})
	}
}

func TestGShadowEntryString(t *testing.T) {
	testCases := []struct {
		description string
		entry       GShadowEntry
		expected    string
	}{
		{
			description: "With admins and users",
			entry: GShadowEntry{
				Groupname: "testgroup",
				Password:  "!!",
				Admins:    []string{"admin1", "admin2"},
				Users:     []string{"user1", "user2"},
			},
			expected: "testgroup:!!:admin1,admin2:user1,user2",
		},
		{
			description: "Empty admins and users",
			entry: GShadowEntry{
				Groupname: "testgroup",
				Password:  "!!",
				Admins:    []string{},
				Users:     []string{},
			},
			expected: "testgroup:!!::",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.entry.String())
		})
	}
}

// Test helper functions

func TestNextID(t *testing.T) {
	testCases := []struct {
		description string
		entries     map[uint16]string
		min         uint16
		max         uint16
		wantID      uint16
		wantErr     error
	}{
		{
			description: "Find first available ID",
			entries:     map[uint16]string{},
			min:         100,
			max:         105,
			wantID:      100,
			wantErr:     nil,
		},
		{
			description: "Skip used IDs",
			entries: map[uint16]string{
				100: "used",
				101: "used",
			},
			min:     100,
			max:     105,
			wantID:  102,
			wantErr: nil,
		},
		{
			description: "No available IDs",
			entries: map[uint16]string{
				100: "used",
				101: "used",
				102: "used",
			},
			min:     100,
			max:     102,
			wantID:  0,
			wantErr: ErrNoAvailableIDs,
		},
		{
			description: "Last available ID",
			entries: map[uint16]string{
				100: "used",
				101: "used",
			},
			min:     100,
			max:     102,
			wantID:  102,
			wantErr: nil,
		},
		{
			description: "All IDs in range are used",
			entries: map[uint16]string{
				1000: "used",
				1001: "used",
				1002: "used",
				1003: "used",
			},
			min:     1000,
			max:     1003,
			wantID:  0,
			wantErr: ErrNoAvailableIDs,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			id, err := nextID(tc.entries, tc.min, tc.max)
			assert.Equal(t, tc.wantErr, err)
			if err == nil {
				assert.Equal(t, tc.wantID, id)
			}
		})
	}
}

func TestNonEmptyStrings(t *testing.T) {
	testCases := []struct {
		description string
		input       []string
		expected    []string
	}{
		{
			description: "All non-empty",
			input:       []string{"a", "b", "c"},
			expected:    []string{"a", "b", "c"},
		},
		{
			description: "Some empty",
			input:       []string{"a", "", "b", "", "c"},
			expected:    []string{"a", "b", "c"},
		},
		{
			description: "All empty",
			input:       []string{"", "", ""},
			expected:    []string{},
		},
		{
			description: "Empty slice",
			input:       []string{},
			expected:    []string{},
		},
		{
			description: "Whitespace not filtered",
			input:       []string{" ", "  ", "a"},
			expected:    []string{" ", "  ", "a"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result := nonEmptyStrings(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHasEntry(t *testing.T) {
	entries := []string{"alice", "bob", "charlie"}

	assert.True(t, hasEntry(entries, "alice"))
	assert.True(t, hasEntry(entries, "bob"))
	assert.True(t, hasEntry(entries, "charlie"))
	assert.False(t, hasEntry(entries, "dave"))
	assert.False(t, hasEntry(entries, ""))
	assert.False(t, hasEntry([]string{}, "alice"))
}

func TestFileExists(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create a file
	afero.WriteFile(fs, "/exists", []byte("content"), 0644)

	exists, err := fileExists(fs, "/exists")
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = fileExists(fs, "/not-exists")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestWriteLines(t *testing.T) {
	fs := afero.NewMemMapFs()

	entries := []PasswdEntry{
		{Username: "root", Password: "x", UID: 0, GID: 0, Comment: "root", HomeDir: "/root", Shell: "/bin/bash"},
		{Username: "bin", Password: "x", UID: 1, GID: 1, Comment: "bin", HomeDir: "/bin", Shell: "/sbin/nologin"},
	}

	err := writeLines(fs, "/tmp/passwd", entries, 0644)
	assert.NoError(t, err)

	content, err := afero.ReadFile(fs, "/tmp/passwd")
	assert.NoError(t, err)

	expected := "root:x:0:0:root:/root:/bin/bash\nbin:x:1:1:bin:/bin:/sbin/nologin\n"
	assert.Equal(t, expected, string(content))

	// Verify temp file was cleaned up
	exists, _ := fileExists(fs, "/tmp/passwd+")
	assert.False(t, exists)
}

func TestWriteLinesEmptyList(t *testing.T) {
	fs := afero.NewMemMapFs()

	entries := []PasswdEntry{}
	err := writeLines(fs, "/tmp/passwd", entries, 0644)
	assert.NoError(t, err)

	content, err := afero.ReadFile(fs, "/tmp/passwd")
	assert.NoError(t, err)
	assert.Equal(t, "", string(content))
}

func TestCreateHomeDir(t *testing.T) {
	fs := afero.NewMemMapFs()

	err := createHomeDir(fs, "/home/testuser", 1000, 1000)
	assert.NoError(t, err)

	// Check home directory exists
	info, err := fs.Stat("/home/testuser")
	assert.NoError(t, err)
	assert.True(t, info.IsDir())

	// Check .ssh directory exists with correct permissions
	sshInfo, err := fs.Stat("/home/testuser/.ssh")
	assert.NoError(t, err)
	assert.True(t, sshInfo.IsDir())
	assert.Equal(t, os.FileMode(0700), sshInfo.Mode().Perm())
}

func TestCreateHomeDirNestedPath(t *testing.T) {
	fs := afero.NewMemMapFs()

	err := createHomeDir(fs, "/home/users/testuser", 1000, 1000)
	assert.NoError(t, err)

	// Check nested parent directory was created
	info, err := fs.Stat("/home/users")
	assert.NoError(t, err)
	assert.True(t, info.IsDir())

	// Check home directory exists
	info, err = fs.Stat("/home/users/testuser")
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

// Test edge cases for AddUser variations

func TestAddUserEdgeCases(t *testing.T) {
	testCases := []struct {
		description string
		setup       func(afero.Fs)
		username    string
		groupname   string
		wantErr     error
	}{
		{
			description: "Duplicate user in existing file",
			setup: func(fs afero.Fs) {
				afero.WriteFile(fs, constants.FileEtcPasswd, []byte("testuser:x:1000:1000::/home/testuser:/bin/bash\n"), 0644)
			},
			username:  "testuser",
			groupname: "testuser",
			wantErr:   ErrUsernameExists,
		},
		{
			description: "Empty username",
			setup:       func(fs afero.Fs) {},
			username:    "",
			groupname:   "testgroup",
			wantErr:     ErrUsernameLength,
		},
		{
			description: "Empty groupname",
			setup:       func(fs afero.Fs) {},
			username:    "testuser",
			groupname:   "",
			wantErr:     ErrGroupnameLength,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			tc.setup(fs)

			_, _, err := AddSystemUser(fs, tc.username, tc.groupname, "/nonexistent", "")
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

func TestAddLoginUserWheelGroupIntegration(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Add first login user - should create wheel group
	uid1, gid1, err := AddLoginUser(fs, "alice", "alice", "/home/alice", "/bin/bash", "")
	assert.NoError(t, err)
	assert.Equal(t, uint16(1000), uid1)
	assert.Equal(t, uint16(1000), gid1)

	// Verify wheel group was created
	_, groupByName, _, err := ParseGroup(fs, constants.FileEtcGroup)
	assert.NoError(t, err)
	wheelGroup, ok := groupByName[constants.GroupNameWheel]
	assert.True(t, ok)
	assert.Contains(t, wheelGroup.Users, "alice")

	// Add second login user - should add to existing wheel group
	uid2, gid2, err := AddLoginUser(fs, "bob", "bob", "/home/bob", "/bin/bash", "")
	assert.NoError(t, err)
	assert.Equal(t, uint16(1001), uid2)
	assert.Equal(t, uint16(1001), gid2)

	// Verify both users in wheel group
	_, groupByName, _, err = ParseGroup(fs, constants.FileEtcGroup)
	assert.NoError(t, err)
	wheelGroup = groupByName[constants.GroupNameWheel]
	assert.Contains(t, wheelGroup.Users, "alice")
	assert.Contains(t, wheelGroup.Users, "bob")
}
