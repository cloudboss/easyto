package login

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/cloudboss/easyto/preinit/constants"
	"github.com/spf13/afero"
)

var (
	ErrUsernameLength  = errors.New("username must be longer than 0")
	ErrGroupnameLength = errors.New("group name must be longer than 0")
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
}

func (g GroupEntry) String() string {
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
}

func (g GShadowEntry) String() string {
	return fmt.Sprintf("%s:%s:%s:%s",
		g.Groupname, g.Password, strings.Join(g.Admins, ","), strings.Join(g.Users, ","))
}

func ParsePasswd(fs afero.Fs, passwdFile string) (map[uint16]*PasswdEntry, map[string]*PasswdEntry, error) {
	f, err := fs.Open(passwdFile)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to open %s: %w", passwdFile, err)
	}
	defer f.Close()

	entryMapUID := make(map[uint16]*PasswdEntry)
	entryMapName := make(map[string]*PasswdEntry)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		fields := strings.Split(line, ":")
		if len(fields) != 7 {
			return nil, nil, fmt.Errorf("unexpected number of fields in %s: %d",
				passwdFile, len(fields))
		}

		uid, err := strconv.ParseUint(fields[2], 10, 16)
		if err != nil {
			return nil, nil, fmt.Errorf("error parsing third field of line in %s: %w",
				passwdFile, err)
		}

		gid, err := strconv.ParseUint(fields[3], 10, 16)
		if err != nil {
			return nil, nil, fmt.Errorf("error parsing fourth field of line in %s: %w",
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
	}

	if err = scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("unable to read %s: %w", passwdFile, err)
	}

	return entryMapUID, entryMapName, nil
}

func ParseGroup(fs afero.Fs, groupFile string) (map[uint16]*GroupEntry, map[string]*GroupEntry, error) {
	f, err := fs.Open(groupFile)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to open %s: %w", groupFile, err)
	}
	defer f.Close()

	entryMapGID := make(map[uint16]*GroupEntry)
	entryMapName := make(map[string]*GroupEntry)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		fields := strings.Split(line, ":")
		if len(fields) != 4 {
			return nil, nil, fmt.Errorf("unexpected number of fields in %s: %d",
				groupFile, len(fields))
		}

		gid, err := strconv.ParseUint(fields[2], 10, 16)
		if err != nil {
			return nil, nil, fmt.Errorf("error parsing third field of line in %s: %w",
				groupFile, err)
		}

		groupEntry := &GroupEntry{
			Groupname: fields[0],
			Password:  fields[1],
			GID:       uint16(gid),
			Users:     nonEmptyStrings(strings.Split(fields[3], ",")),
		}
		entryMapGID[uint16(gid)] = groupEntry
		entryMapName[groupEntry.Groupname] = groupEntry
	}

	if err = scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("unable to read %s: %w", groupFile, err)
	}

	return entryMapGID, entryMapName, nil
}

func ParseShadow(fs afero.Fs, shadowFile string) (map[string]ShadowEntry, error) {
	f, err := fs.Open(shadowFile)
	if err != nil {
		return nil, fmt.Errorf("unable to open %s: %w", shadowFile, err)
	}
	defer f.Close()

	entryMap := make(map[string]ShadowEntry)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		fields := strings.Split(line, ":")
		if len(fields) != 9 {
			return nil, fmt.Errorf("unexpected number of fields in %s: %d",
				shadowFile, len(fields))
		}

		lastChange := -1
		if len(fields[2]) > 0 {
			lastChange, err = strconv.Atoi(fields[2])
			if err != nil {
				return nil, fmt.Errorf("error parsing third field of line in %s: %w",
					shadowFile, err)
			}
		}

		minAge := -1
		if len(fields[3]) > 0 {
			minAge, err = strconv.Atoi(fields[3])
			if err != nil {
				return nil, fmt.Errorf("error parsing fourth field of line in %s: %w",
					shadowFile, err)
			}
		}

		maxAge := -1
		if len(fields[4]) > 0 {
			maxAge, err = strconv.Atoi(fields[4])
			if err != nil {
				return nil, fmt.Errorf("error parsing fifth field of line in %s: %w",
					shadowFile, err)
			}
		}

		warningPeriod := -1
		if len(fields[5]) > 0 {
			warningPeriod, err = strconv.Atoi(fields[5])
			if err != nil {
				return nil, fmt.Errorf("error parsing sixth field of line in %s: %w",
					shadowFile, err)
			}
		}

		inactivityPeriod := -1
		if len(fields[6]) > 0 {
			inactivityPeriod, err = strconv.Atoi(fields[6])
			if err != nil {
				return nil, fmt.Errorf("error parsing seventh field of line in %s: %w",
					shadowFile, err)
			}
		}

		expiration := -1
		if len(fields[7]) > 0 {
			expiration, err = strconv.Atoi(fields[7])
			if err != nil {
				return nil, fmt.Errorf("error parsing eighth field of line in %s: %w",
					shadowFile, err)
			}
		}

		username := fields[0]
		entryMap[username] = ShadowEntry{
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
	}

	if err = scanner.Err(); err != nil {
		return nil, fmt.Errorf("unable to read %s: %w", shadowFile, err)
	}

	return entryMap, nil
}

func ParseGShadow(fs afero.Fs, shadowFile string) (map[string]GShadowEntry, error) {
	f, err := fs.Open(shadowFile)
	if err != nil {
		return nil, fmt.Errorf("unable to open %s: %w", shadowFile, err)
	}
	defer f.Close()

	entryMap := make(map[string]GShadowEntry)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		fields := strings.Split(line, ":")
		if len(fields) != 4 {
			return nil, fmt.Errorf("unexpected number of fields in %s: %d",
				shadowFile, len(fields))
		}

		groupname := fields[0]
		entryMap[groupname] = GShadowEntry{
			Groupname: groupname,
			Password:  fields[1],
			Admins:    nonEmptyStrings(strings.Split(fields[2], ",")),
			Users:     nonEmptyStrings(strings.Split(fields[3], ",")),
		}
	}

	if err = scanner.Err(); err != nil {
		return nil, fmt.Errorf("unable to read %s: %w", shadowFile, err)
	}

	return entryMap, nil
}

func nextID[T any](entries map[uint16]T, start uint16) (uint16, error) {
	const max uint16 = 1<<15 - 1
	id := start
	for {
		if _, ok := entries[id]; !ok {
			return id, nil
		}
		if id >= max {
			return 0, fmt.Errorf("no available IDs")
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

func addFileEntry(fs afero.Fs, path, line string, mode os.FileMode) error {
	if _, err := fs.Stat(path); os.IsNotExist(err) {
		return addFileEntryNew(fs, path, line, mode)
	}
	return addFileEntryExisting(fs, path, line, mode)
}

func addFileEntryNew(fs afero.Fs, path, line string, mode os.FileMode) error {
	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)

	f, err := fs.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, mode)
	if err != nil {
		return fmt.Errorf("unable to open %s: %w", path, err)
	}
	defer f.Close()

	if _, err = f.WriteString(line + "\n"); err != nil {
		return fmt.Errorf("unable to write to %s: %w", path, err)
	}

	return nil
}

func addFileEntryExisting(fs afero.Fs, path, line string, mode os.FileMode) error {
	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)

	tmpPath := path + "+"
	tf, err := fs.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY, mode)
	if err != nil {
		return fmt.Errorf("unable to open %s: %w", tmpPath, err)
	}
	// File will normally be closed earlier if there are no errors.
	defer tf.Close()

	f, err := fs.Open(path)
	if err != nil {
		return fmt.Errorf("unable to open %s: %w", path, err)
	}
	// As above, file will be closed earlier if there are no errors.
	defer f.Close()

	_, err = io.Copy(tf, f)
	if err != nil {
		return fmt.Errorf("unable to copy %s to %s: %w", path, tmpPath, err)
	}

	if _, err = tf.WriteString(line + "\n"); err != nil {
		return fmt.Errorf("unable to write to %s: %w", tmpPath, err)
	}

	if err = tf.Close(); err != nil {
		return fmt.Errorf("unable to close %s: %w", tmpPath, err)
	}

	if err = f.Close(); err != nil {
		return fmt.Errorf("unable to close %s: %w", path, err)
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
func AddSystemUser(fs afero.Fs, username, groupname, homeDir string) (uint16, uint16, error) {
	return AddUser(fs, username, groupname, homeDir, "/bin/false", 100, false, true)
}

// AddLoginUser adds a user that can log in with a valid shell and home directory.
func AddLoginUser(fs afero.Fs, username, groupname, homeDir string) (uint16, uint16, error) {
	return AddUser(fs, username, groupname, homeDir, "/bin/sh", 1000, true, false)
}

// AddUser adds a user to the system.
func AddUser(fs afero.Fs, username, groupname, homeDir, shell string,
	idStart uint16, createHome, locked bool) (uint16, uint16, error) {
	var (
		addToPasswd         = true
		addToShadow         = true
		addToGroup          = true
		addToGShadow        = true
		uid          uint16 = idStart
		gid          uint16 = idStart
	)

	if len(username) == 0 {
		return 0, 0, ErrUsernameLength
	}

	if len(groupname) == 0 {
		return 0, 0, ErrGroupnameLength
	}

	passwdFileExists, err := fileExists(fs, constants.FileEtcPasswd)
	if err != nil {
		return 0, 0, err
	}
	groupFileExists, err := fileExists(fs, constants.FileEtcGroup)
	if err != nil {
		return 0, 0, err
	}
	shadowFileExists, err := fileExists(fs, constants.FileEtcShadow)
	if err != nil {
		return 0, 0, err
	}
	gShadowFileExists, err := fileExists(fs, constants.FileEtcGShadow)
	if err != nil {
		return 0, 0, err
	}

	if passwdFileExists {
		passwdByUID, passwdByName, err := ParsePasswd(fs, constants.FileEtcPasswd)
		if err != nil {
			return 0, 0, err
		}

		if _, ok := passwdByName[username]; ok {
			addToPasswd = false
		} else {
			uid, err = nextID(passwdByUID, uid)
			if err != nil {
				return 0, 0, err
			}
		}

		if !shadowFileExists {
			addToShadow = false
		} else {
			shadowByName, err := ParseShadow(fs, constants.FileEtcShadow)
			if err != nil {
				return 0, 0, err
			}
			if _, ok := shadowByName[username]; ok {
				addToShadow = false
			}
		}
	}

	if groupFileExists {
		groupByGID, groupByName, err := ParseGroup(fs, constants.FileEtcGroup)
		if err != nil {
			return 0, 0, err
		}

		if _, ok := groupByName[groupname]; ok {
			addToGroup = false
		} else {
			gid, err = nextID(groupByGID, gid)
			if err != nil {
				return 0, 0, err
			}
		}

		if !gShadowFileExists {
			addToGShadow = false
		} else {
			gShadowByName, err := ParseGShadow(fs, constants.FileEtcGShadow)
			if err != nil {
				return 0, 0, err
			}
			if _, ok := gShadowByName[groupname]; ok {
				addToGShadow = false
			}
		}
	}

	if addToPasswd {
		passwdEntry := &PasswdEntry{
			Username: username,
			Password: "x",
			UID:      uid,
			GID:      gid,
			Comment:  username,
			HomeDir:  homeDir,
			Shell:    shell,
		}
		err := addFileEntry(fs, constants.FileEtcPasswd, passwdEntry.String(), constants.ModeEtcPasswd)
		if err != nil {
			return 0, 0, err
		}
	}

	if addToGroup {
		groupEntry := &GroupEntry{
			Groupname: groupname,
			Password:  "x",
			GID:       gid,
			Users:     []string{username},
		}
		err := addFileEntry(fs, constants.FileEtcGroup, groupEntry.String(), constants.ModeEtcGroup)
		if err != nil {
			return 0, 0, err
		}
	}

	if addToShadow {
		shadowEntry := ShadowEntry{
			Username:         username,
			Password:         "*",
			LastChange:       0,
			MinAge:           0,
			MaxAge:           99999,
			WarningPeriod:    7,
			InactivityPeriod: -1,
			Expiration:       -1,
		}
		if locked {
			shadowEntry.Password = "!!"
		}
		err := addFileEntry(fs, constants.FileEtcShadow, shadowEntry.String(), constants.ModeEtcShadow)
		if err != nil {
			return 0, 0, err
		}
	}

	if addToGShadow {
		gShadowEntry := GShadowEntry{
			Groupname: groupname,
			Password:  "!!",
			Admins:    []string{},
			Users:     []string{username},
		}
		err := addFileEntry(fs, constants.FileEtcGShadow, gShadowEntry.String(), constants.ModeEtcGShadow)
		if err != nil {
			return 0, 0, err
		}
	}

	if createHome {
		err := createHomeDir(fs, homeDir, uid, gid)
		if err != nil {
			return 0, 0, err
		}
	}

	return uid, gid, nil
}
