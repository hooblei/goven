
package goven


import (
    "fmt"
    "os"
    "time"
    "io"
    "io/ioutil"
    "strings"
    "bufio"
    "regexp"
    "path/filepath"
    "github.com/russross/blackfriday"
)


var (
    // Mon Jan 2 15:04:05 MST 2006
    DT_FORMATS = []string{
        time.RFC3339,
        "2006-01-02T15:04:05",
        "2006-01-02 15:04:05",
        "2006-01-02T15:04",
        "02.01.2006 15:04:05",
        "2.1.2006 15:04:05",
        "02.01.2006 15:04",
        "2.1.2006 15:04",
        "2006-01-02 15:04",
        "2006-01-02",
        "02.01.2006",
        "2.1.2006",
        // tbc ...
    }
)


type Post struct {
    Path string
    Author string
    Cdt time.Time
    Headers map[string]string
    Title string
    BodyIdx int
}

func (p *Post) Read(r io.Reader) {

    type St uint8

    const (
        S_BODY = St(1 << iota)
        S_HEADER
        S_START = St(0)
    )

    hdr := regexp.MustCompile(`^\s?(?P<name>\w+):\s?(?P<value>.+)\s?`)
    hdrsep := regexp.MustCompile(`^\s?[-\*]{3,}\s?$`)
    titlere := regexp.MustCompile(`^#{1,8} (?P<title>.+)\s?`)

    // XXX: Allow multiline headers, titles etc?
    rdr := bufio.NewReader(r)
    lbuf := []string{}
    st := S_START
    idx := 0

    loop: for {
        var ln string
        var err error

        if ln, err = rdr.ReadString(0x0a); err != nil && err != io.EOF {
            return
        }

        switch {
            // Enable header mode if first line matches heder sep
            case st == S_START && hdrsep.MatchString(ln):
                st = S_HEADER
            // Found header row
            case st == S_HEADER && hdr.MatchString(ln):
                lbuf = append(lbuf, ln[:len(ln) - 1])
            // Header delimiter
            case st == S_HEADER && hdrsep.MatchString(ln):
                for _, l := range lbuf {
                    m := hdr.FindStringSubmatch(l)
                    p.Headers[strings.ToLower(m[1])] = m[2]
                }
                if t, ok := p.Headers["title"]; ok {
                    p.Title = strings.TrimSpace(t)
                }
                lbuf = []string{}
                p.BodyIdx = idx + len(ln)
                st = S_BODY
            // Non-Header row found - reset to body body mode
            case st == S_HEADER && !hdr.MatchString(ln):
                p.BodyIdx = 0
                lbuf = []string{}
                st = S_BODY
            // Scan body until we find a title
            case st <= S_BODY && p.Title == "" && titlere.MatchString(ln):
                p.Title = strings.TrimSpace(titlere.FindStringSubmatch(ln)[1])
                st = S_BODY
            // Mode body and title found - nothing left todo ... for now
            case st == S_BODY && p.Title != "":
                break loop
        }

        idx += len(ln)

        if err == io.EOF {
            break loop
        }
    }

    for key, val := range p.Headers {
        switch {
            case key == "created":
                if t, e := str2time(val, DT_FORMATS); e == nil {
                    p.Cdt = t
                } else {
                    fmt.Println("TPE", e)
                }
            case key == "author":
                p.Author = val
        }
    }

    return
}

// Returns the Post body
func (p *Post) Body() (buf []byte, err error) {
    if buf, err = ioutil.ReadFile(p.Path); err != nil {
        return
    }

    buf = buf[p.BodyIdx:]

    return
}

// Render Post contents into HTML assuming that the post is valid Markdown
func (p *Post) RenderBody() (buf []byte, err error) {
    if buf, err = p.Body(); err != nil {
        return
    }

    buf = blackfriday.MarkdownCommon(buf)

    return
}

// Returns the 'slugified' Post title
func (p *Post) Slug() string {
    return Slugified(p.Title)
}

// Create a new post instance from given `path`
func NewPost(path string) *Post {
    var err error
    var finfo os.FileInfo
    var file *os.File

    if finfo, err = os.Stat(path); err != nil {
        return nil
    }

    p := &Post{
        Path: path,
        Headers: make(map[string]string),
        Cdt: finfo.ModTime(),
    }

    if file, err = os.Open(path); err == nil {
        defer file.Close()
        p.Read(file)
    }

    return p
}


type Posts []*Post

func (p Posts) Len() int {
    return len(p)
}

func (p Posts) Swap(i, j int) {
    p[i], p[j] = p[j], p[i]
}

func NewPosts(sourcePath string) Posts {
    var err error
    var posts = Posts{}

    walk := func (p string, fi os.FileInfo, e error) (err error) {
        if e != nil {
            return
        }

        if p == sourcePath || fi.IsDir() {
            return
        }

        posts = append(posts, NewPost(p))

        return
    }

    if err = filepath.Walk(sourcePath, walk); err != nil {
        return nil
    }

    return posts
}


type PostsByCdt struct {
    Posts
}

func (p PostsByCdt) Less(i, j int) bool {
    return p.Posts[i].Cdt.Unix() < p.Posts[j].Cdt.Unix()
}

