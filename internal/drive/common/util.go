package common

import (
	"fmt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	gdrive "google.golang.org/api/drive/v3"
)

const DirectoryMimeType = "application/vnd.google-apps.folder"

func IsDir(f *gdrive.File) bool {
	return f.MimeType == DirectoryMimeType
}

func IsBinary(f *gdrive.File) bool {
	return f.Md5Checksum != ""
}

func FormatList(a []string) string {
	return strings.Join(a, ", ")
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

func FormatBool(b bool) string {
	return cases.Title(language.English).String(strconv.FormatBool(b))
}

func FormatDatetime(iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	local := t.Local()
	year, month, day := local.Date()
	hour, min, sec := local.Clock()
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d", year, month, day, hour, min, sec)
}

// Truncates string to given max length, and inserts ellipsis into
// the middle of the string to signify that the string has been truncated
func TruncateString(str string, maxRunes int) string {
	indicator := "..."

	// Number of runes in string
	runeCount := utf8.RuneCountInString(str)

	// Return input string if length of input string is less than max length
	// Input string is also returned if max length is less than 9 which is the minmal supported length
	if runeCount <= maxRunes || maxRunes < 9 {
		return str
	}

	// Number of remaining runes to be removed
	remaining := (runeCount - maxRunes) + utf8.RuneCountInString(indicator)

	var truncated string
	var skip bool

	for leftOffset, char := range str {
		rightOffset := runeCount - (leftOffset + remaining)

		// Start skipping chars when the left and right offsets are equal
		// Or in the case where we wont be able to do an even split: when the left offset is larger than the right offset
		if leftOffset == rightOffset || (leftOffset > rightOffset && !skip) {
			skip = true
			truncated += indicator
		}

		if skip && remaining > 0 {
			// Skip char and decrement the remaining skip counter
			remaining--
			continue
		}

		// Add char to result string
		truncated += string(char)
	}

	// Return truncated string
	return truncated
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return false
}

func Mkdir(path string) error {
	dir := filepath.Dir(path)
	if FileExists(dir) {
		return nil
	}
	return os.MkdirAll(dir, 0775)
}

func IntMax() int64 {
	return 1<<(strconv.IntSize-1) - 1
}

func PathLength(path string) int {
	return strings.Count(path, string(os.PathSeparator))
}

func ParentFilePath(path string) string {
	dir, _ := filepath.Split(path)
	return filepath.Dir(dir)
}

func Pow(x int, y int) int {
	f := math.Pow(float64(x), float64(y))
	return int(f)
}

func Min(x int, y int) int {
	n := math.Min(float64(x), float64(y))
	return int(n)
}

func OpenFile(path string) (*os.File, os.FileInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to open file: %s", err)
	}

	info, err := f.Stat()
	if err != nil {
		return nil, nil, fmt.Errorf("Failed getting file metadata: %s", err)
	}

	return f, info, nil
}
