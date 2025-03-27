package modapi

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
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
	enabledModsFile = "enabled.txt"
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

type listFile struct {
	file *os.File
	list []string
}

func newListFile(path string) (listFile, error) {
	file, err := os.Open(path)
	if err != nil {
		file.Close()
		return listFile{}, err
	}

	content, err := io.ReadAll(file)
	if err != nil {
		file.Close()
		return listFile{}, err
	}

	return listFile{file: file, list: strings.Split(strings.TrimSpace(string(content)), "\n")}, nil
}

func (lf *listFile) Close() error {
	defer lf.file.Close()

	_, err := lf.file.WriteString(strings.Join(lf.list, "\n"))
	return err
}

func (lf listFile) checkIndex(i int) {
	if i >= lf.Len() {
		panic(fmt.Sprintf("%d is not a valid index"))
	}
}

func (lf listFile) Len() int {
	return len(lf.list)
}

func (lf listFile) Has(s string) bool {
	return slices.Contains(lf.list, s)
}

func (lf listFile) Index(s string) int {
	return slices.Index(lf.list, s)
}

func (lf *listFile) Push(s string) {
	lf.list = append(lf.list, s)
}

func (lf *listFile) Add(i int, s string) {
	lf.checkIndex(i)

	lf.list = append(append(lf.list[:i], s), lf.list[i:]...)
}

func (lf *listFile) Pop() {
	lf.Remove(lf.Len())
}

func (lf *listFile) Remove(i int) {
	lf.checkIndex(i)

	lf.list = append(lf.list[:i], lf.list[i+1:]...)
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
		))
		if err != nil {
			return err
		}

		err = func() error {
			enabledMods, err := os.Create(enabledModsFile)
			enabledMods.Close()
			return err
		}()
		if err != nil {
			return err
		}
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
}

type ModEntry struct {
	enabled bool
	name    string
}

func (me ModEntry) String() string {
	if me.enabled {
		return "[ENABLED] " + me.name
	}

	return "[DISABLED] " + me.name
}

type ModHandler struct {
	localModHandler
	hh          HomeHandler
	pluginsPath string
	enabledMods listFile
}

func NewModHandler(hh HomeHandler) (ModHandler, error) {
	lmh := localModHandler{}

	_, err := toml.DecodeFile(hh.Path(rushFolder, configFile), &lmh)
	if err != nil {
		return ModHandler{}, err
	}

	enabledMods, err := newListFile(hh.Path(rushFolder, enabledModsFile))
	if err != nil {
		enabledMods.Close()
		return ModHandler{}, err
	}

	mh := ModHandler{localModHandler: lmh, enabledMods: enabledMods, hh: hh, pluginsPath: path.Join(lmh.hastePath, "BepInEx", "plugins")}
	if ok, err := doesFileExist(mh.pluginsPath); err != nil {
		return ModHandler{}, err
	} else if !ok {
		return ModHandler{}, errors.New("BepInEx is not installed, please install it to use Rush")
	}

	return mh, nil
}

func (mh ModHandler) path(subpath ...string) string {
	return path.Join(append([]string{mh.modsPath}, subpath...)...)
}

func (mh ModHandler) Config() string {
	return mh.hh.Path(rushFolder, configFile)
}

func (mh ModHandler) Close() error {
	return mh.enabledMods.Close()
}

func (mh ModHandler) PathsOfMod(name string) (string, string) {
	return path.Join(mh.modsPath, name+".dll"), path.Join(mh.modsPath, name+".pdb")
}

func (mh ModHandler) PathsOfPlugin(name string) (string, string) {
	return path.Join(mh.pluginsPath, name+".dll"), path.Join(mh.pluginsPath, name+".pdb")
}

func (mh ModHandler) GetMods() ([]ModEntry, error) {
	dir, err := os.ReadDir(mh.modsPath)
	if err != nil {
		return []ModEntry{}, err
	}

	entries := []ModEntry{}

	for _, m := range dir {
		s := strings.Split(m.Name(), ".")
		if s[1] == ".dll" {
			entries = append(entries, ModEntry{name: s[0], enabled: mh.enabledMods.Has(s[0])})
		}
	}

	return entries, nil
}

func (mh ModHandler) InstallMods(modpaths ...string) error {
	for _, mp := range modpaths {
		if err := mh.InstallMod(mp); err != nil {
			return err
		}
	}

	return nil
}

func (mh ModHandler) InstallMod(modpath string) error {
	ext := path.Ext(modpath)
	if ext == ".zip" {
		z, err := zip.OpenReader(modpath)
		defer z.Close()
		if err != nil {
			return err
		}

		for _, f := range z.File {
			ext = path.Ext(f.Name)
			if ext == ".dll" || ext == ".pdb" {
				fr, err := f.OpenRaw()
				if err != nil {
					return err
				}

				content, err := io.ReadAll(fr)
				if err != nil {
					return err
				}

				err = func() error {
					out, err := os.Create(mh.path(path.Base(f.Name)))
					defer out.Close()
					if err != nil {
						return err
					}

					_, err = out.Write(content)
					return err
				}()
				if err != nil {
					return err
				}
			}
		}
	} else if ext == ".dll" || ext == ".pdb" {
		content, err := os.ReadFile(modpath)
		if err != nil {
			return err
		}

		out, err := os.Create(mh.path(path.Base(modpath)))
		defer out.Close()
		if err != nil {
			return err
		}

		_, err = out.Write(content)
		if err != nil {
			return err
		}
	}

	return fmt.Errorf("'%s' is not an installable file", path.Base(modpath))
}

func (mh *ModHandler) EnableMods(names ...string) error {
	for _, name := range names {
		if err := mh.EnableMod(name); err != nil {
			return err
		}
	}

	return nil
}

func (mh *ModHandler) EnableMod(name string) error {
	dllPath, pdbPath := mh.PathsOfMod(name)

	dllContent, err := os.ReadFile(dllPath)
	if err != nil {
		return err
	}

	err = func() error {
		out, err := os.Create(path.Join(mh.pluginsPath, name+".dll"))
		defer out.Close()
		if err != nil {
			return err
		}

		_, err = out.Write(dllContent)
		return err
	}()
	if err != nil {
		return err
	}

	pdbContent, err := os.ReadFile(pdbPath)
	if err != nil {
		if !isFileNotFound(err, pdbPath) {
			return nil
		}

		return err
	}

	return func() error {
		out, err := os.Create(path.Join(mh.pluginsPath, name+".pdb"))
		defer out.Close()
		if err != nil {
			return err
		}

		_, err = out.Write(pdbContent)
		return err
	}()
}

func (mh *ModHandler) DisableMods(names ...string) error {
	for _, name := range names {
		if err := mh.DisableMod(name); err != nil {
			return err
		}
	}

	return nil
}

func (mh *ModHandler) DisableMod(name string) error {
	i := mh.enabledMods.Index(name)
	if i == -1 {
		return nil
	}

	mh.enabledMods.Remove(i)

	dllPath, pdbPath := mh.PathsOfPlugin(name)

	err := os.Remove(dllPath)
	if err != nil {
		return err
	}

	err = os.Remove(pdbPath)
	if isFileNotFound(err, name) {
		return nil
	}

	return err
}
