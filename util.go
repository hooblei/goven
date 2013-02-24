
package goven

import (
    "strings"
    "time"
    "regexp"
    "github.com/sloonz/go-iconv"
)


// Create a 'slugified' version of given input string `ipt`
func Slugified(ipt string) string {
    var out = ""
    var err error

    if out, err = iconv.Conv(ipt, "ASCII//IGNORE", "UTF-8"); err == nil {
        out = regexp.MustCompile(`\W+`).ReplaceAllString(out, "-")
        out = regexp.MustCompile(`-{2,}`).ReplaceAllString(out, "-")
        out = strings.Trim(out, "-")
    }

    return out
}


// Try to parse given string `s` into `time.Time` using datetime `formats`
func str2time(s string, formats []string) (t time.Time, err error) {
    for _, f := range formats {
        if t, err = time.Parse(f, s); err == nil {
            return
        }
    }

    return
}

