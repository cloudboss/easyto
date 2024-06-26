package login

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/cloudboss/easyto/pkg/constants"
	"github.com/spf13/afero"
)

var (
	ErrNoAvailableIDs  = errors.New("no available IDs")
	ErrUsernameExists  = errors.New("username exists")
	ErrUsernameLength  = errors.New("username must be longer than 0")
	ErrGroupnameLength = errors.New("group name must be longer than 0")
)

const (
	UID_GID_MIN            = 1000
	UID_GID_MAX     uint16 = 1<<15 - 1
	UID_GID_MIN_SYS        = 100
	UID_GID_MAX_SYS        = 999
)

type PasswdEntry struct {
	Username string
	Password string
	UID      uint16
	GID      uint16
	Comment  string
	HomeDir  string
	Shell    string
}

func (p PasswdEntry) String() string {
	return fmt.Sprintf("%s:%s:%d:%d:%s:%s:%s",
		p.Username, p.Password, p.UID, p.GID, p.Comment, p.HomeDir, p.Shell)
}

type GroupEntry struct {
	Groupname string
	Password  string
	GID       uint16
	Users     []string
	index     int
}

func (g *GroupEntry) String() string {
	return fmt.Sprintf("%s:%s:%d:%s",
		g.Groupname, g.Password, g.GID, strings.Join(g.Users, ","))
}

type ShadowEntry struct {
	Username         string
	Password         string
	LastChange       int
	MinAge           int
	MaxAge           int
	WarningPeriod    int
	InactivityPeriod int
	Expiration       int
	Unused           string
}

func (s ShadowEntry) String() string {
	syscall.Getpid()
	inactivityPeriod := ""
	if s.InactivityPeriod >= 0 {
		inactivityPeriod = strconv.Itoa(s.InactivityPeriod)
	}
	lastChange := ""
	if s.LastChange >= 0 {
		lastChange = strconv.Itoa(s.LastChange)
	}
	expiration := ""
	if s.Expiration >= 0 {
		expiration = strconv.Itoa(s.Expiration)
	}
	minAge := ""
	if s.MinAge >= 0 {
		minAge = strconv.Itoa(s.MinAge)
	}
	maxAge := ""
	if s.MaxAge >= 0 {
		maxAge = strconv.Itoa(s.MaxAge)
	}
	warningPeriod := ""
	if s.WarningPeriod >= 0 {
		warningPeriod = strconv.Itoa(s.WarningPeriod)
	}
	return fmt.Sprintf("%s:%s:%s:%s:%s:%s:%s:%s:%s",
		s.Username, s.Password, lastChange, minAge, maxAge,
		warningPeriod, inactivityPeriod, expiration, s.Unused)
}

type GShadowEntry struct {
	Groupname string
	Password  string
	Admins    []string
	Users     []string
	index     int
}

func (g GShadowEntry) String() string {
	return fmt.Sprintf("%s:%s:%s:%s",
		g.Groupname, g.Password, strings.Join(g.Admins, ","), strings.Join(g.Users, ","))
}

func ParsePasswd(fs afero.Fs, passwdFile string) (map[uint16]*PasswdEntry, map[string]*PasswdEntry, []*PasswdEntry, error) {
	f, err := fs.Open(passwdFile)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("unable to open %s: %w", passwdFile, err)
	}
	defer f.Close()

	entryMapUID := make(map[uint16]*PasswdEntry)
	entryMapName := make(map[string]*PasswdEntry)
	entryList := []*PasswdEntry{}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		fields := strings.Split(line, ":")
		if len(fields) != 7 {
			return nil, nil, nil, fmt.Errorf("unexpected number of fields in %s: %d",
				passwdFile, len(fields))
		}

		uid, err := strconv.ParseUint(fields[2], 10, 16)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("error parsing third field of line in %s: %w",
				passwdFile, err)
		}

		gid, err := strconv.ParseUint(fields[3], 10, 16)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("error parsing fourth field of line in %s: %w",
				passwdFile, err)
		}

		pwent := &PasswdEntry{
			Username: fields[0],
			Password: fields[1],
			UID:      uint16(uid),
			GID:      uint16(gid),
			Comment:  fields[4],
			HomeDir:  fields[5],
			Shell:    fields[6],
		}
		entryMapUID[uint16(uid)] = pwent
		entryMapName[pwent.Username] = pwent
		entryList = append(entryList, pwent)
	}

	if err = scanner.Err(); err != nil {
		return nil, nil, nil, fmt.Errorf("unable to read %s: %w", passwdFile, err)
	}

	return entryMapUID, entryMapName, entryList, nil
}

func ParseGroup(fs afero.Fs, groupFile string) (map[uint16]*GroupEntry, map[string]*GroupEntry, []*GroupEntry, error) {
	f, err := fs.Open(groupFile)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("unable to open %s: %w", groupFile, err)
	}
	defer f.Close()

	entryMapGID := make(map[uint16]*GroupEntry)
	entryMapName := make(map[string]*GroupEntry)
	entryList := []*GroupEntry{}

	i := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		fields := strings.Split(line, ":")
		if len(fields) != 4 {
			return nil, nil, nil, fmt.Errorf("unexpected number of fields in %s: %d",
				groupFile, len(fields))
		}

		gid, err := strconv.ParseUint(fields[2], 10, 16)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("error parsing third field of line in %s: %w",
				groupFile, err)
		}

		groupEntry := &GroupEntry{
			Groupname: fields[0],
			Password:  fields[1],
			GID:       uint16(gid),
			Users:     nonEmptyStrings(strings.Split(fields[3], ",")),
			index:     i,
		}
		entryMapGID[uint16(gid)] = groupEntry
		entryMapName[groupEntry.Groupname] = groupEntry
		entryList = append(entryList, groupEntry)
		i++
	}

	if err = scanner.Err(); err != nil {
		return nil, nil, nil, fmt.Errorf("unable to read %s: %w", groupFile, err)
	}

	return entryMapGID, entryMapName, entryList, nil
}

func ParseShadow(fs afero.Fs, shadowFile string) (map[string]*ShadowEntry, []*ShadowEntry, error) {
	f, err := fs.Open(shadowFile)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to open %s: %w", shadowFile, err)
	}
	defer f.Close()

	entryMap := make(map[string]*ShadowEntry)
	entryList := []*ShadowEntry{}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		fields := strings.Split(line, ":")
		if len(fields) != 9 {
			return nil, nil, fmt.Errorf("unexpected number of fields in %s: %d",
				shadowFile, len(fields))
		}

		lastChange := -1
		if len(fields[2]) > 0 {
			lastChange, err = strconv.Atoi(fields[2])
			if err != nil {
				return nil, nil, fmt.Errorf("error parsing third field of line in %s: %w",
					shadowFile, err)
			}
		}

		minAge := -1
		if len(fields[3]) > 0 {
			minAge, err = strconv.Atoi(fields[3])
			if err != nil {
				return nil, nil, fmt.Errorf("error parsing fourth field of line in %s: %w",
					shadowFile, err)
			}
		}

		maxAge := -1
		if len(fields[4]) > 0 {
			maxAge, err = strconv.Atoi(fields[4])
			if err != nil {
				return nil, nil, fmt.Errorf("error parsing fifth field of line in %s: %w",
					shadowFile, err)
			}
		}

		warningPeriod := -1
		if len(fields[5]) > 0 {
			warningPeriod, err = strconv.Atoi(fields[5])
			if err != nil {
				return nil, nil, fmt.Errorf("error parsing sixth field of line in %s: %w",
					shadowFile, err)
			}
		}

		inactivityPeriod := -1
		if len(fields[6]) > 0 {
			inactivityPeriod, err = strconv.Atoi(fields[6])
			if err != nil {
				return nil, nil, fmt.Errorf("error parsing seventh field of line in %s: %w",
					shadowFile, err)
			}
		}

		expiration := -1
		if len(fields[7]) > 0 {
			expiration, err = strconv.Atoi(fields[7])
			if err != nil {
				return nil, nil, fmt.Errorf("error parsing eighth field of line in %s: %w",
					shadowFile, err)
			}
		}

		username := fields[0]
		entry := &ShadowEntry{
			Username:         username,
			Password:         fields[1],
			LastChange:       lastChange,
			MinAge:           minAge,
			MaxAge:           maxAge,
			WarningPeriod:    warningPeriod,
			InactivityPeriod: inactivityPeriod,
			Expiration:       expiration,
			Unused:           fields[8],
		}
		entryMap[username] = entry
		entryList = append(entryList, entry)
	}

	if err = scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("unable to read %s: %w", shadowFile, err)
	}

	return entryMap, entryList, nil
}

func ParseGShadow(fs afero.Fs, shadowFile string) (map[string]*GShadowEntry, []*GShadowEntry, error) {
	f, err := fs.Open(shadowFile)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to open %s: %w", shadowFile, err)
	}
	defer f.Close()

	entryMap := make(map[string]*GShadowEntry)
	entryList := []*GShadowEntry{}

	i := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		fields := strings.Split(line, ":")
		if len(fields) != 4 {
			return nil, nil, fmt.Errorf("unexpected number of fields in %s: %d",
				shadowFile, len(fields))
		}

		groupname := fields[0]
		entry := &GShadowEntry{
			Groupname: groupname,
			Password:  fields[1],
			Admins:    nonEmptyStrings(strings.Split(fields[2], ",")),
			Users:     nonEmptyStrings(strings.Split(fields[3], ",")),
			index:     i,
		}
		entryMap[groupname] = entry
		entryList = append(entryList, entry)
		i++
	}

	if err = scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("unable to read %s: %w", shadowFile, err)
	}

	return entryMap, entryList, nil
}

func nextID[T any](entries map[uint16]T, min, max uint16) (uint16, error) {
	id := min
	for {
		if _, ok := entries[id]; !ok {
			return id, nil
		}
		if id >= max {
			return 0, ErrNoAvailableIDs
		}
		id++
	}
}

func nonEmptyStrings(strs []string) []string {
	nonEmpty := []string{}
	for _, s := range strs {
		if len(s) > 0 {
			nonEmpty = append(nonEmpty, s)
		}
	}
	return nonEmpty
}

func fileExists(fs afero.Fs, path string) (bool, error) {
	if _, err := fs.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("unable to stat %s: %w", path, err)
	}
	return true, nil
}

func writeLines[T fmt.Stringer](fs afero.Fs, path string, lines []T, mode os.FileMode) error {
	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)

	tmpPath := path + "+"
	tf, err := fs.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY, mode)
	if err != nil {
		return fmt.Errorf("unable to open %s: %w", tmpPath, err)
	}
	// File will normally be closed earlier if there are no errors.
	defer tf.Close()

	for _, line := range lines {
		if _, err = tf.WriteString(fmt.Sprintf("%s\n", line)); err != nil {
			return fmt.Errorf("unable to write to %s: %w", tmpPath, err)
		}

	}

	if err = tf.Close(); err != nil {
		return fmt.Errorf("unable to close %s: %w", tmpPath, err)
	}

	err = fs.Rename(tmpPath, path)
	if err != nil {
		return fmt.Errorf("unable to rename %s to %s: %w", tmpPath, path, err)
	}

	return nil
}

func createHomeDir(fs afero.Fs, homeDir string, uid, gid uint16) error {
	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)

	parent := filepath.Dir(homeDir)
	sshDir := filepath.Join(homeDir, ".ssh")

	if err := fs.MkdirAll(parent, 0755); err != nil {
		return fmt.Errorf("unable to create %s: %w", parent, err)
	}

	if err := fs.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("unable to create %s: %w", sshDir, err)
	}

	if err := fs.Chown(homeDir, int(uid), int(gid)); err != nil {
		return fmt.Errorf("unable to change ownership of %s: %w", homeDir, err)
	}

	if err := fs.Chown(sshDir, int(uid), int(gid)); err != nil {
		return fmt.Errorf("unable to change ownership of %s: %w", sshDir, err)
	}

	return nil
}

// AddSystemUser adds a system user with no password or valid shell.
func AddSystemUser(fs afero.Fs, username, groupname, homeDir, baseDir string) (uint16, uint16, error) {
	return AddUser(fs, username, groupname, homeDir, "/bin/false", baseDir, UID_GID_MIN_SYS,
		UID_GID_MAX_SYS, false, false, true)
}

// AddLoginUser adds a user that can log in with a valid shell and home directory.
func AddLoginUser(fs afero.Fs, username, groupname, homeDir, shell, baseDir string) (uint16, uint16, error) {
	return AddUser(fs, username, groupname, homeDir, shell, baseDir, UID_GID_MIN, UID_GID_MAX,
		true, true, false)
}

// AddRootUser adds the root user.
func AddRootUser(fs afero.Fs, shell, baseDir string) (uint16, uint16, error) {
	homeDir := filepath.Join(constants.DirETRoot, "/root")
	return AddUser(fs, "root", "root", homeDir, shell, baseDir, 0, 0, true, false, false)
}

// AddUser adds a user to the system.
func AddUser(fs afero.Fs, username, groupname, homeDir, shell, baseDir string,
	idMin, idMax uint16, createHome, isLoginUser, locked bool) (uint16, uint16, error) {
	var (
		createShadowEntry  = true
		createGroupEntry   = true
		createGShadowEntry = true
		modifiedGroup      = false
		modifiedShadow     = false
		modifiedGShadow    = false
		passwdByUID        map[uint16]*PasswdEntry
		passwdByName       map[string]*PasswdEntry
		passwdList         []*PasswdEntry
		shadowByName       map[string]*ShadowEntry
		shadowList         []*ShadowEntry
		groupByGID         map[uint16]*GroupEntry
		groupByName        map[string]*GroupEntry
		groupList          []*GroupEntry
		gShadowByName      map[string]*GShadowEntry
		gShadowList        []*GShadowEntry
		uid                uint16 = idMin
		gid                uint16 = idMin
		idStartWheel       uint16 = 10
	)

	if len(username) == 0 {
		return 0, 0, ErrUsernameLength
	}

	if len(groupname) == 0 {
		return 0, 0, ErrGroupnameLength
	}

	fileEtcPasswd := filepath.Join(baseDir, constants.FileEtcPasswd)
	fileEtcGroup := filepath.Join(baseDir, constants.FileEtcGroup)
	fileEtcShadow := filepath.Join(baseDir, constants.FileEtcShadow)
	fileEtcGShadow := filepath.Join(baseDir, constants.FileEtcGShadow)

	passwdFileExists, err := fileExists(fs, fileEtcPasswd)
	if err != nil {
		return 0, 0, err
	}
	groupFileExists, err := fileExists(fs, fileEtcGroup)
	if err != nil {
		return 0, 0, err
	}
	shadowFileExists, err := fileExists(fs, fileEtcShadow)
	if err != nil {
		return 0, 0, err
	}
	gShadowFileExists, err := fileExists(fs, fileEtcGShadow)
	if err != nil {
		return 0, 0, err
	}

	if passwdFileExists {
		passwdByUID, passwdByName, passwdList, err = ParsePasswd(fs, fileEtcPasswd)
		if err != nil {
			return 0, 0, err
		}

		if _, ok := passwdByName[username]; ok {
			return 0, 0, ErrUsernameExists
		} else {
			uid, err = nextID(passwdByUID, uid, idMax)
			if err != nil {
				return 0, 0, err
			}
		}

		if !shadowFileExists {
			createShadowEntry = false
		} else {
			shadowByName, shadowList, err = ParseShadow(fs, fileEtcShadow)
			if err != nil {
				return 0, 0, err
			}
			if _, ok := shadowByName[username]; ok {
				createShadowEntry = false
			}
		}
	}

	if groupFileExists {
		groupByGID, groupByName, groupList, err = ParseGroup(fs, fileEtcGroup)
		if err != nil {
			return 0, 0, err
		}

		if _, ok := groupByName[groupname]; ok {
			createGroupEntry = false
		} else {
			gid, err = nextID(groupByGID, gid, idMax)
			if err != nil {
				return 0, 0, err
			}
		}

		if !gShadowFileExists {
			createGShadowEntry = false
		} else {
			gShadowByName, gShadowList, err = ParseGShadow(fs, fileEtcGShadow)
			if err != nil {
				return 0, 0, err
			}
			if _, ok := gShadowByName[groupname]; ok {
				createGShadowEntry = false
			}
		}
	}

	if isLoginUser {
		wheelGroup, ok := groupByName[constants.GroupNameWheel]
		if ok {
			i := wheelGroup.index
			if !hasEntry(groupList[i].Users, username) {
				groupList[i].Users = append(groupList[i].Users, username)
				modifiedGroup = true
			}
		} else {
			wheelGID, err := nextID(groupByGID, idStartWheel, UID_GID_MAX_SYS)
			if err != nil {
				return 0, 0, fmt.Errorf("unable to get next GID for %s: %w",
					constants.GroupNameWheel, err)
			}
			wheelGroup = &GroupEntry{
				Groupname: constants.GroupNameWheel,
				Password:  "x",
				GID:       wheelGID,
				Users:     []string{username},
			}
			groupList = append(groupList, wheelGroup)
			modifiedGroup = true
		}

		if createGShadowEntry {
			wheelGShadow, ok := gShadowByName[constants.GroupNameWheel]
			if ok {
				i := wheelGShadow.index
				if !hasEntry(gShadowList[i].Users, username) {
					gShadowList[i].Users = append(gShadowList[i].Users, username)
					modifiedGShadow = true
				}
			} else {
				wheelGShadow = &GShadowEntry{
					Groupname: constants.GroupNameWheel,
					Users:     []string{username},
				}
				gShadowList = append(gShadowList, wheelGShadow)
				modifiedGShadow = true
			}
		}
	}

	passwdEntry := &PasswdEntry{
		Username: username,
		Password: "x",
		UID:      uid,
		GID:      gid,
		Comment:  username,
		HomeDir:  homeDir,
		Shell:    shell,
	}
	passwdList = append(passwdList, passwdEntry)

	err = writeLines(fs, fileEtcPasswd, passwdList, constants.ModeEtcPasswd)
	if err != nil {
		return 0, 0, err
	}

	if createGroupEntry {
		groupEntry := &GroupEntry{
			Groupname: groupname,
			Password:  "x",
			GID:       gid,
			Users:     []string{username},
		}
		groupList = append(groupList, groupEntry)
		modifiedGroup = true
	}

	if createShadowEntry {
		shadowEntry := &ShadowEntry{
			Username:         username,
			Password:         "*",
			LastChange:       0,
			MinAge:           0,
			MaxAge:           99999,
			WarningPeriod:    7,
			InactivityPeriod: -1,
			Expiration:       -1,
		}
		if isLoginUser {
			shadowEntry.LastChange = -1
		}
		if locked {
			shadowEntry.Password = "!!"
		}
		shadowList = append(shadowList, shadowEntry)
		modifiedShadow = true
	}

	if createGShadowEntry {
		gShadowEntry := &GShadowEntry{
			Groupname: groupname,
			Password:  "!!",
			Admins:    []string{},
			Users:     []string{username},
		}
		gShadowList = append(gShadowList, gShadowEntry)
		modifiedGShadow = true
	}

	if modifiedGroup {
		err := writeLines(fs, fileEtcGroup, groupList, constants.ModeEtcGroup)
		if err != nil {
			return 0, 0, err
		}
	}

	if modifiedShadow {
		err := writeLines(fs, fileEtcShadow, shadowList, constants.ModeEtcShadow)
		if err != nil {
			return 0, 0, err
		}
	}

	if modifiedGShadow {
		err := writeLines(fs, fileEtcGShadow, gShadowList, constants.ModeEtcGShadow)
		if err != nil {
			return 0, 0, err
		}
	}

	if createHome {
		err := createHomeDir(fs, filepath.Join(baseDir, homeDir), uid, gid)
		if err != nil {
			return 0, 0, err
		}
	}

	return uid, gid, nil
}

func hasEntry(entries []string, entry string) bool {
	for _, e := range entries {
		if e == entry {
			return true
		}
	}
	return false
}
