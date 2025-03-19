//go:build linux

package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"
)

const ranFile = "/var/run/fixids.ran"

var logger = log.New(os.Stderr, "", 0)
var quietFlag = flag.Bool("q", false, "quiet mode")

func main() {
	runtime.GOMAXPROCS(1)
	logger.SetPrefix("fixids: ")
	flag.Parse()

	argsWithoutProg := flag.Args()

	if len(argsWithoutProg) < 2 {
		logger.Fatalln("Usage: fixids user-name group-name")
	}

	// detect what user we are running as
	runtimeUIDInt := os.Getuid()
	runtimeUID := strconv.Itoa(runtimeUIDInt)
	runtimeGIDInt := os.Getgid()
	runtimeGID := strconv.Itoa(runtimeGIDInt)

	// only run once on the system
	if _, err := os.Stat(ranFile); !os.IsNotExist(err) {
		logInfo("already ran on this system; will not attempt to change UID/GID")
		exitOrExec(runtimeUID, runtimeUIDInt, runtimeGIDInt, -1, argsWithoutProg)
	}

	// check that script is running as root
	if os.Geteuid() != 0 {
		logger.Fatalln(`fixids is not running as root, ensure that the following criteria are met:
        - fixids binary is owned by root: 'chown root:root /path/to/fixids'
        - fixids binary has the setuid bit: 'chmod u+s /path/to/fixids'
        - NoNewPrivileges is disabled in container security profile
        - volume containing fixids binary does not have the 'nosuid' mount option`)
	}

	// validate the container user from the config
	containerUser := argsWithoutProg[0]

	containerUID, containerUIDError := findUID(containerUser)
	if containerUIDError != nil {
		logger.Fatalln(containerUIDError)
	}
	if containerUID == "" {
		logger.Fatalln("user '" + containerUser + "' does not exist")
	}
	containerUIDInt, err := strconv.Atoi(containerUID)
	if err != nil {
		logger.Fatal(err)
	}
	containerUIDUint32 := uint32(containerUIDInt) // #nosec G115

	containerGroup := argsWithoutProg[1]
	containerGID, containerGIDError := findGID(containerGroup)
	if containerGIDError != nil {
		logger.Fatalln(containerGIDError)
	}
	if containerGID == "" {
		logger.Fatalln("group '" + containerGroup + "' does not exist")
	}
	containerGIDInt, err := strconv.Atoi(containerGID)
	if err != nil {
		logger.Fatal(err)
	}
	containerGIDUint32 := uint32(containerGIDInt) // #nosec G115

	argsWithoutProg = argsWithoutProg[2:]

	home, err := findHomeDir(containerUID)
	if err != nil {
		logger.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		logger.Fatal(err)
	}

	paths := []string{home, cwd}

	// declare uid/gid vars and
	var oldUID, newUID, oldGID, newGID string
	needChown := false

	// decide if need to change UIDs
	existingUser, existingUserError := findUser(runtimeUID)
	if existingUserError != nil {
		logger.Fatalln(existingUserError)
	}
	if existingUser == "" {
		logInfo("updating user '" + containerUser + "' to UID '" + runtimeUID + "'")
		needChown = true
		oldUID = containerUID
		newUID = runtimeUID
	} else {
		oldUID = ""
		newUID = ""
		if existingUser == containerUser {
			logInfo("runtime UID '" + runtimeUID + "' already matches container user '" + containerUser + "' UID")
		} else {
			logInfo("runtime UID '" + runtimeUID + "' matches existing user '" + existingUser + "'; not changing UID")
			needChown = true
		}
	}

	// decide if need to change GIDs
	existingGroup, existingGroupError := findGroup(runtimeGID)
	if existingGroupError != nil {
		logger.Fatalln(existingGroupError)
	}
	if existingGroup == "" {
		logInfo("updating group '" + containerGroup + "' to GID '" + runtimeGID + "'")
		needChown = true
		oldGID = containerGID
		newGID = runtimeGID
	} else {
		oldGID = ""
		newGID = ""
		if existingGroup == containerGroup {
			logInfo("runtime GID '" + runtimeGID + "' already matches container group '" + containerGroup + "' GID")
		} else {
			logInfo("runtime GID '" + runtimeGID + "' matches existing group '" + existingGroup + "'; not changing GID")
			needChown = true
		}
	}

	// update /etc/passwd if necessary
	if oldUID != newUID || oldGID != newGID {
		err := updateEtcPasswd(containerUser, oldUID, newUID, oldGID, newGID)
		if err != nil {
			logger.Fatalln(err)
		}
	}

	// update /etc/group if necessary
	if oldGID != newGID {
		err := updateEtcGroup(containerGroup, oldGID, newGID)
		if err != nil {
			logger.Fatalln(err)
		}
	}

	// search entire filesystem and chown containerUID:containerGID to runtimeUID:runtimeGID
	if needChown {

		// process /proc/mounts
		mounts, err := parseProcMounts()
		if err != nil {
			logger.Fatalln(err)
		}

		// store the current mountpoint
		var mountpoint string

		// this function is called for every file visited
		visit := func(filePath string, fileInfo os.FileInfo, err error) error {

			// an error to lstat or filepath.readDirNames
			// see https://github.com/boxboat/fixids/issues/4
			if err != nil {
				logInfo("error when visiting " + filePath)
				logInfo(err)
				return nil
			}

			// stat file to determine UID and GID
			sys, ok := fileInfo.Sys().(*syscall.Stat_t)
			if !ok {
				logInfo("cannot stat " + filePath)
				return filepath.SkipDir
			}

			// prevent recursing into mounts
			if findMountpoint(filePath, mounts) != mountpoint {
				if sys.Uid == containerUIDUint32 && sys.Gid == containerGIDUint32 {
					logInfo("skipping mounted path " + filePath)
				}
				if fileInfo.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// only chown if file is containerUID:containerGID
			if sys.Uid == containerUIDUint32 && sys.Gid == containerGIDUint32 {
				logInfo("chown " + filePath)
				err := syscall.Lchown(filePath, runtimeUIDInt, runtimeGIDInt)
				if err != nil {
					logInfo("error changing owner of " + filePath)
					logInfo(err)
				}
				return nil
			}
			return nil
		}

		for _, path := range paths {
			// stat the path to ensure it exists
			_, err := os.Stat(path)
			if err != nil {
				logInfo("error accessing path: " + path)
				logInfo(err)
				continue
			}
			mountpoint = findMountpoint(path, mounts)

			logInfo("recursively searching path " + path)
			if err := filepath.Walk(path, visit); err != nil {
				logInfo("error searching path: " + path)
				logInfo(err)
			}
		}

	}

	// mark the script as ran
	err = os.WriteFile(ranFile, []byte{}, 0644) // #nosec G306
	if err != nil {
		logger.Fatalln(err)
	}

	// if the existing HOME directory is "/", change it to the user's home directory
	existingHomeDir := os.Getenv("HOME")
	if existingHomeDir == "/" {
		homeDir, homeDirErr := findHomeDir(runtimeUID)
		if homeDirErr == nil && homeDir != "" && homeDir != "/" {
			if len(argsWithoutProg) > 0 {
				if err := os.Setenv("HOME", homeDir); err != nil {
					logger.Fatalln(err)
				}
			} else {
				fmt.Println(`export HOME="` + strings.Replace(homeDir, `"`, `\"`, -1) + `"`)
			}
		}
	}

	oldGIDInt := -1
	if oldGID != "" && oldGID != newGID {
		if gid, err := strconv.Atoi(oldGID); err != nil {
			oldGIDInt = gid
		}
	}

	// all done
	exitOrExec(runtimeUID, runtimeUIDInt, runtimeGIDInt, oldGIDInt, argsWithoutProg)
}

func logInfo(v ...interface{}) {
	if !*quietFlag {
		logger.Println(v...)
	}
}

// oldGIDInt should be -1 if the GID was not changed
func exitOrExec(runtimeUID string, runtimeUIDInt, runtimeGIDInt, oldGIDInt int, argsWithoutProg []string) {
	if len(argsWithoutProg) > 0 {
		// exec mode - de-escalate privileges and exec new process
		binary, err := exec.LookPath(argsWithoutProg[0])
		if err != nil {
			logger.Fatalln(err)
		}

		// get real user
		user, err := findUser(runtimeUID)
		if err != nil {
			logger.Fatalln(err)
		}

		// set groups
		if user != "" {
			// get all existing group IDs
			existingGIDs, err := syscall.Getgroups()
			if err != nil {
				logger.Fatalln(err)
			}

			// get primary GID from /etc/passwd
			primaryGID, err := findPrimaryGID(runtimeUID)
			if err != nil {
				logger.Fatalln(err)
			}

			// get supplementary GIDs from /etc/group
			supplementaryGIDs, err := findUserSupplementaryGIDs(user)
			if err != nil {
				logger.Fatalln(err)
			}

			// add all GIDs to a map
			allGIDs := append(existingGIDs, primaryGID)
			allGIDs = append(allGIDs, supplementaryGIDs...)
			gidMap := make(map[int]struct{})
			for _, gid := range allGIDs {
				gidMap[gid] = struct{}{}
			}

			// remove the old GID if it was changed
			if oldGIDInt >= 0 {
				delete(gidMap, oldGIDInt)
			}

			groups := make([]int, 0, len(gidMap))
			for gid := range gidMap {
				groups = append(groups, gid)
			}

			// set groups
			err = syscall.Setgroups(groups)
			if err != nil {
				logger.Fatalln(err)
			}
		}

		// de-escalate the group back to the original
		if err := syscall.Setegid(runtimeGIDInt); err != nil {
			logger.Fatalln(err)
		}

		// de-escalate the user back to the original
		if err := syscall.Seteuid(runtimeUIDInt); err != nil {
			logger.Fatalln(err)
		}

		// exec new process
		env := os.Environ()
		err = syscall.Exec(binary, argsWithoutProg, env) // #nosec G204
		if err != nil {
			logger.Fatalln(err)
		}
	}

	// nothing to exec; exit the program
	os.Exit(0)
}

func searchColonDelimitedFile(filePath string, search string, searchOffset int, returnOffset int) (string, error) {
	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		cols := strings.Split(scanner.Text(), ":")
		if len(cols) < (searchOffset+1) || len(cols) < (returnOffset+1) {
			continue
		}
		if cols[searchOffset] == search {
			return cols[returnOffset], nil
		}
	}
	return "", nil
}

func findUID(user string) (string, error) {
	return searchColonDelimitedFile("/etc/passwd", user, 0, 2)
}

func findUser(uid string) (string, error) {
	return searchColonDelimitedFile("/etc/passwd", uid, 2, 0)
}

// returns -1 if not found
func findPrimaryGID(uid string) (int, error) {
	gid, err := searchColonDelimitedFile("/etc/passwd", uid, 2, 3)
	if err != nil {
		return -1, err
	}
	if gid == "" {
		return -1, nil
	}
	return strconv.Atoi(gid)
}

func findHomeDir(uid string) (string, error) {
	return searchColonDelimitedFile("/etc/passwd", uid, 2, 5)
}

func findGID(group string) (string, error) {
	return searchColonDelimitedFile("/etc/group", group, 0, 2)
}

func findGroup(gid string) (string, error) {
	return searchColonDelimitedFile("/etc/group", gid, 2, 0)
}

func findUserSupplementaryGIDs(user string) ([]int, error) {
	// group:pass:gid:users
	file, err := os.Open("/etc/group")
	if err != nil {
		return nil, err
	}

	var gids []int
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		cols := strings.Split(scanner.Text(), ":")
		if len(cols) < 4 {
			continue
		}
		users := strings.Split(cols[3], ",")
		if !slices.Contains(users, user) {
			continue
		}
		gid, err := strconv.Atoi(cols[2])
		if err != nil {
			continue
		}
		gids = append(gids, gid)
	}

	if err := file.Close(); err != nil {
		return nil, err
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return gids, nil
}

func updateEtcPasswd(user string, oldUID string, newUID string, oldGID string, newGID string) error {
	// user:pass:uid:gid:comment:home dir:shell
	file, err := os.Open("/etc/passwd")
	if err != nil {
		return err
	}

	newLines := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		cols := strings.Split(scanner.Text(), ":")
		if len(cols) < 4 {
			continue
		}
		if oldUID != "" && newUID != "" && cols[0] == user && cols[2] == oldUID {
			cols[2] = newUID
		}
		if oldGID != "" && newGID != "" && cols[3] == oldGID {
			cols[3] = newGID
		}
		newLines += strings.Join(cols, ":") + "\n"
	}

	if err := file.Close(); err != nil {
		return err
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	err = os.WriteFile("/etc/passwd", []byte(newLines), 0644) // #nosec G306
	if err != nil {
		return err
	}

	return nil
}

func updateEtcGroup(group string, oldGID string, newGID string) error {
	// group:pass:gid:users
	file, err := os.Open("/etc/group")
	if err != nil {
		return err
	}

	newLines := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		cols := strings.Split(scanner.Text(), ":")
		if len(cols) < 3 {
			continue
		}
		if oldGID != "" && newGID != "" && cols[0] == group && cols[2] == oldGID {
			cols[2] = newGID
		}
		newLines += strings.Join(cols, ":") + "\n"
	}

	if err := file.Close(); err != nil {
		return err
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	err = os.WriteFile("/etc/group", []byte(newLines), 0644) // #nosec G306
	if err != nil {
		return err
	}

	return nil
}

func parseProcMounts() (map[string]bool, error) {
	// device mountpoint type options dump fsck
	// spaces appear as \040
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}

	mounts := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		cols := strings.Fields(scanner.Text())
		if len(cols) >= 2 {
			mounts[filepath.Clean(strings.Replace(cols[1], "\\040", " ", -1))] = true
		}
	}

	if err := file.Close(); err != nil {
		return nil, err
	}

	return mounts, nil
}

func findMountpoint(path string, mounts map[string]bool) string {
	path = filepath.Clean(path)
	var lastPath string
	for path != lastPath {
		if _, ok := mounts[path]; ok {
			return path
		}
		lastPath = path
		path = filepath.Dir(path)
	}
	return "/"
}
