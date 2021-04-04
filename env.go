package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/keys-pub/keys/env"
	"github.com/pkg/errors"
)

// Env for app runtime.
// Do not store anything sensitive in here, values are saved clear and can be
// modified at will.
// Env is not authenticated.
type Env struct {
	appName string
	build   Build
	values  map[string]string
	linkDir string
}

// NewEnv loads the Env.
func NewEnv(appName string, build Build) (*Env, error) {
	if appName == "" {
		return nil, errors.Errorf("no app name")
	}
	env := &Env{
		appName: appName,
		build:   build,
		linkDir: filepath.Join("usr", "local", "bin"),
	}
	if err := env.Load(); err != nil {
		return nil, err
	}
	return env, nil
}

// Env key names
const keysPubServerCfgKey = "keys-pub-server"
const chillServerCfgKey = "chill-server"
const portCfgKey = "port"

var configKeys = []string{keysPubServerCfgKey, chillServerCfgKey, portCfgKey}

// IsKey returns true if config key is recognized.
func (e Env) IsKey(s string) bool {
	for _, k := range configKeys {
		if s == k {
			return true
		}
	}
	return false
}

// Port to connect.
func (e Env) Port() int {
	return e.GetInt(portCfgKey, e.build.DefaultPort)
}

// KeysPubServerURL to connect to.
func (e Env) KeysPubServerURL() string {
	return e.Get(keysPubServerCfgKey, "https://keys.pub")
}

// ChillServerURL to connect to.
func (e Env) ChillServerURL() string {
	return e.Get(chillServerCfgKey, "https://getchill.app")
}

// Build describes build flags.
type Build struct {
	Version        string
	Commit         string
	Date           string
	DefaultAppName string
	DefaultPort    int
	ServiceName    string
	CmdName        string
	Description    string
}

func (b Build) String() string {
	return fmt.Sprintf("%s %s %s", b.Version, b.Commit, b.Date)
}

// AppName returns current app name.
func (e Env) AppName() string {
	return e.appName
}

// AppDir is where app related files are persisted.
func (e Env) AppDir() string {
	p, err := e.AppPath("", false)
	if err != nil {
		panic(err)
	}
	return p
}

// LogsDir is where logs are written.
func (e Env) LogsDir() string {
	p, err := e.LogsPath("", false)
	if err != nil {
		panic(err)
	}
	return p
}

// AppPath ...
func (e Env) AppPath(file string, makeDir bool) (string, error) {
	opts := []env.PathOption{env.Dir(e.AppName()), env.File(file)}
	if makeDir {
		opts = append(opts, env.Mkdir())
	}
	return env.AppPath(opts...)
}

// LogsPath ...
func (e Env) LogsPath(file string, makeDir bool) (string, error) {
	opts := []env.PathOption{env.Dir(e.AppName()), env.File(file)}
	if makeDir {
		opts = append(opts, env.Mkdir())
	}
	return env.LogsPath(opts...)
}

func (e Env) certPath(makeDir bool) (string, error) {
	return e.AppPath("ca.pem", makeDir)
}

// Path to config file.
func (e *Env) Path(makeDir bool) (string, error) {
	return e.AppPath("config.json", makeDir)
}

// Load ...
func (e *Env) Load() error {
	path, err := e.Path(false)
	if err != nil {
		return err
	}

	var values map[string]string

	exists, err := pathExists(path)
	if err != nil {
		return err
	}
	if exists {
		b, err := ioutil.ReadFile(path) // #nosec
		if err != nil {
			return err
		}
		if err := json.Unmarshal(b, &values); err != nil {
			return err
		}
	}
	if values == nil {
		values = map[string]string{}
	}
	e.values = values
	return nil
}

// Save ...
func (e *Env) Save() error {
	path, err := e.Path(true)
	if err != nil {
		return err
	}
	b, err := json.Marshal(e.values)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(path, b, filePerms); err != nil {
		return err
	}
	return nil
}

// Reset removes saved values.
func (e *Env) Reset() error {
	path, err := e.Path(false)
	if err != nil {
		return err
	}

	exists, err := pathExists(path)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	return os.Remove(path)
}

// Export ...
func (e Env) Export() ([]byte, error) {
	return json.MarshalIndent(e.values, "", "  ")
}

// Get config value.
func (e *Env) Get(key string, dflt string) string {
	v, ok := e.values[key]
	if !ok {
		return dflt
	}
	return v
}

// GetInt gets config value as int.
func (e *Env) GetInt(key string, dflt int) int {
	v, ok := e.values[key]
	if !ok {
		return dflt
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		logger.Warningf("config value %s not an int", key)
		return 0
	}
	return n

}

// GetBool gets config value as bool.
func (e *Env) GetBool(key string) bool {
	v, ok := e.values[key]
	if !ok {
		return false
	}
	b, _ := truthy(v)
	return b
}

// SetBool sets bool value for key.
func (e *Env) SetBool(key string, b bool) {
	e.Set(key, truthyString(b))
}

// SetInt sets int value for key.
func (e *Env) SetInt(key string, n int) {
	e.Set(key, strconv.Itoa(n))
}

// Set value.
func (e *Env) Set(key string, value string) {
	e.values[key] = value
}

func (e *Env) savePortFlag(port int) error {
	if port == 0 {
		return nil
	}
	e.SetInt(portCfgKey, port)
	return e.Save()
}

func truthy(s string) (bool, error) {
	s = strings.TrimSpace(s)
	switch s {
	case "1", "t", "true", "y", "yes":
		return true, nil
	case "0", "f", "false", "n", "no":
		return false, nil
	default:
		return false, errors.Errorf("invalid value: %s", s)
	}
}

func truthyString(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
