package constants

const (
	DirProc = "/proc"

	FileEtcPasswd  = "/etc/passwd"
	FileEtcShadow  = "/etc/shadow"
	FileEtcGroup   = "/etc/group"
	FileEtcGShadow = "/etc/gshadow"

	FileMetadata = "metadata.json"

	GroupNameWheel = "wheel"

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

	DirETRoot string
	DirETBin  = DirETRoot + "/bin"
	DirETSbin = DirETRoot + "/sbin"
	DirETHome = DirETRoot + "/home"
)
