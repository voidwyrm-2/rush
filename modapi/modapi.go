package modapi

import (
	"fmt"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	hasteFolderName = "Haste Broken Worlds Demo"
	rushFolder      = "rushmm"
	configFile      = "config.toml"
)

/*
Warning: this calls `os.Open`, so try to use `isFileNotFound` instead when possible
*/
func doesFileExist(path string) (bool, error) {
	f, err := os.Open(path)
	defer f.Close()
	if !isFileNotFound(err, path) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func isFileNotFound(err error, file string) bool {
	if err == nil {
		return false
	}

	s := strings.Split(err.Error(), ":")
	if len(s) < 3 {
		return false
	}

	if err.Error() == "Error: open "+file+": no such file or directory" {
		return true
	} else {
		return strings.HasPrefix(s[1], "open ") && strings.HasSuffix(s[2], file+": no such file or directory")
	}
}

type HomeHandler struct {
	home string
}

func NewHomeHandler() (HomeHandler, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return HomeHandler{}, err
	}

	return HomeHandler{home: home}, nil
}

func (hh HomeHandler) Home() string {
	return hh.home
}

func (hh HomeHandler) Path(subpath ...string) string {
	return path.Join(append([]string{hh.home}, subpath...)...)
}

func (hh HomeHandler) VerifyRushFolder() error {
	p := hh.Path(rushFolder)
	if dir, err := os.ReadDir(p); isFileNotFound(err, p) || (err == nil && len(dir) == 0) {
		return hh.InitRushFolder()
	} else if err != nil {
		return err
	}

	return nil
}

func (hh HomeHandler) InitRushFolder() error {
	rushf := hh.Path(rushFolder)

	ok, err := doesFileExist(rushf)
	if err != nil {
		return err
	} else if !ok {
		if err = os.Mkdir(rushf, os.ModeDir); err != nil {
			return err
		}
	}

	configf := path.Join(rushf, configFile)

	if ok {
		ok, err = doesFileExist(configf)
		if err != nil {
			return err
		}
	}

	if !ok {
		conf, err := os.Create(configf)
		defer conf.Close()
		if err != nil {
			return err
		}

		hastePath, err := ResolveHastePath()
		if err != nil {
			return err
		}

		_, err = conf.WriteString(fmt.Sprintf(`modsPath = "%s"
hastePath = "%s"`,
			path.Join(rushf, "mods"),
			hastePath,
		),
		)
		return err
	}

	modsf := path.Join(rushf, "mods")

	ok, err = doesFileExist(modsf)
	if err != nil {
		return err
	} else if !ok {
		if err = os.Mkdir(modsf, os.ModeDir); err != nil {
			return err
		}
	}

	return nil
}

type localModHandler struct {
	modsPath, hastePath string
	enabledMods         []string
}

type ModEntry struct {
	name    string
	enabled bool
}

func (me ModEntry) String() string {
	if me.enabled {
		return "[ENABLED] " + me.name
	}

	return "[DISABLED] " + me.name
}

type ModHandler struct {
	localModHandler
	hh HomeHandler
}

func NewModHandler(hh HomeHandler) (ModHandler, error) {
	mh := localModHandler{}

	_, err := toml.DecodeFile(hh.Path(rushFolder, configFile), &mh)
	if err != nil {
		return ModHandler{}, err
	}

	return ModHandler{localModHandler: mh, hh: hh}, nil
}

func (mh ModHandler) GetMods() ([]ModEntry, error) {
	dir, err := os.ReadDir(mh.modsPath)
	if err != nil {
		return []ModEntry{}, err
	}

	entries := []ModEntry{}

	for _, m := range dir {
		s := strings.Split(m.Name(), ".")
		if s[1] == "pdb" {
			entries = append(entries, ModEntry{name: s[0], enabled: slices.Contains(mh.enabledMods, s[0])})
		}
	}

	return entries, nil
}
