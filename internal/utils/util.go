package utils

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

func GetDefaultConfigDir() string {
	return filepath.Join(Homedir(), ".config", "gdrive")
}

func ConfigFilePath(basePath, name string) string {
	return filepath.Join(basePath, name)
}

func Homedir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	if runtime.GOOS == "windows" {
		return os.Getenv("USERPROFILE")
	}
	return os.Getenv("HOME")
}

func Equal(a, b []string) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func ExitF(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	fmt.Println("")
	os.Exit(1)
}

func CheckErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func WriteJSON(path string, data interface{}) error {
	tmpFile := path + ".tmp"
	f, err := os.Create(tmpFile)
	if err != nil {
		return err
	}

	err = json.NewEncoder(f).Encode(data)
	f.Close()
	if err != nil {
		os.Remove(tmpFile)
		return err
	}

	return os.Rename(tmpFile, path)
}

const SecretFilename = "secret.json"

type AccountSecret struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func accountSecretPath(basePath string) string {
	return filepath.Join(basePath, SecretFilename)
}

func LoadAccountSecret(basePath string) (AccountSecret, error) {
	content, err := os.ReadFile(accountSecretPath(basePath))
	if err != nil {
		return AccountSecret{}, err
	}

	var secret AccountSecret
	if err := json.Unmarshal(content, &secret); err != nil {
		return AccountSecret{}, err
	}

	return secret, nil
}

func SaveAccountSecret(basePath string, secret AccountSecret) error {
	if err := WriteJSON(accountSecretPath(basePath), secret); err != nil {
		return err
	}
	return os.Chmod(accountSecretPath(basePath), 0600)
}

func Md5sum(path string) string {
	h := md5.New()
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	io.Copy(h, f)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func FormatSize(bytes int64, forceBytes bool) string {
	if bytes == 0 {
		return ""
	}

	if forceBytes {
		return fmt.Sprintf("%v B", bytes)
	}

	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}

	var i int
	value := float64(bytes)

	for value > 1000 {
		value /= 1000
		i++
	}
	return fmt.Sprintf("%.1f %s", value, units[i])
}

func CalcRate(bytes int64, start, end time.Time) int64 {
	seconds := float64(end.Sub(start).Seconds())
	if seconds < 1.0 {
		return bytes
	}
	return round(float64(bytes) / seconds)
}

func round(n float64) int64 {
	if n < 0 {
		return int64(math.Ceil(n - 0.5))
	}
	return int64(math.Floor(n + 0.5))
}
