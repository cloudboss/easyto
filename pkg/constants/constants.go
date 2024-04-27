package constants

const (
	DirCB       = "/__cb__"
	DirHome     = "/__cb__/home"
	DirRun      = "/run"
	DirServices = "/__cb__/services"
	DirVar      = "/var"

	FileEtcPasswd  = "/etc/passwd"
	FileEtcShadow  = "/etc/shadow"
	FileEtcGroup   = "/etc/group"
	FileEtcGShadow = "/etc/gshadow"

	ModeEtcPasswd  = 0644
	ModeEtcShadow  = 0
	ModeEtcGroup   = 0644
	ModeEtcGShadow = 0
)

// "Constants" that are defined with ldflags during compile.
var (
	ChronyUser     string
	SSHPrivsepDir  string
	SSHPrivsepUser string
)
