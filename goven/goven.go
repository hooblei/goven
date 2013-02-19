
package main

import (
    "fmt"
    "io/ioutil"
    "encoding/json"
    "path"
    "os"
    "time"
    "strings"
    "regexp"
    "io"
    "bufio"
    "sort"
    "path/filepath"
    "github.com/russross/blackfriday"
    "text/template"
    "goven"
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



// Try to parse given string `s` into `time.Time` using datetime `formats`
func str2time(s string, formats []string) (t time.Time, err error) {
    for _, f := range formats {
        if t, err = time.Parse(f, s); err == nil {
            return
        }
    }

    return
}


type Config struct {
    Src string `json:"src"`
    Dst string `json:"dst"`
    Theme string `json:"theme"`
    BaseUrl string `json:"baseUrl"`
    Title string `json:"title"`
    Categories []string `json:"categories"`
    Posters []string `json:"posters"`
}


type Theme struct {
    Templates map[string]*template.Template
}

func NewTheme(tpath string) *Theme {
    var err error
    var pattern string
    var pages []string
    var base *template.Template
    var tmap = map[string]*template.Template{}

    pattern = path.Join(tpath, "*.html")
    base = template.Must(template.ParseGlob(pattern))
    pattern = path.Join(tpath, "pages", "*.html")
    if pages, err = filepath.Glob(pattern); err != nil {
        panic(err)
    }

    for _, tpath := range pages {
        var ts *template.Template

        if ts, err = base.Clone(); err != nil {
            panic(err)
        }
        if _, err = ts.ParseFiles(tpath); err != nil {
            panic(err)
        }
        tmap[path.Base(tpath)] = ts
   }

   return &Theme{
       Templates: tmap,
   }
}


type Blog struct {
    Conf *Config
    Title string
    Theme *Theme
}

func (b *Blog) Render(p *Post) (err error) {
    return
}

func NewBlog(conf *Config) *Blog {
    return &Blog{
        Conf: conf,
        Title: conf.Title,
        Theme: NewTheme(conf.Theme),
    }
}


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

func (p *Post) Body() (buf []byte, err error) {
    if buf, err = ioutil.ReadFile(p.Path); err != nil {
        return
    }

    buf = buf[p.BodyIdx:]

    return
}

func (p *Post) RenderBody() (buf []byte, err error) {
    if buf, err = p.Body(); err != nil {
        return
    }

    buf = blackfriday.MarkdownCommon(buf)

    return
}

func (p *Post) Slug() string {
    return goven.Slugified(p.Title)
}

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


type PostsByCdt struct {
    Posts
}

func (p PostsByCdt) Less(i, j int) bool {
    return p.Posts[i].Cdt.Unix() < p.Posts[j].Cdt.Unix()
}

type Page struct {
    Cdt time.Time
    Blog *Blog
    Path string
    Title string
    Content map[string]interface{}
}

func (p *Page) Url() string {
    return path.Join(p.Blog.Conf.BaseUrl, p.Path)
}

func NewPage(blog *Blog, path string, content map[string]interface{}) *Page {
    return &Page{
        Cdt: time.Now(),
        Blog: blog,
        Path: path,
        Content: content,
    }
}


type Index struct {
    Posts Posts
}

// XXX: Make order variable?
func (idx *Index) Pages(blog *Blog) (pages []*Page) {
    sort.Sort(PostsByCdt{idx.Posts})
    for idx, post := range idx.Posts {
        ppath := fmt.Sprintf("%s.html", post.Slug())
        page := NewPage(blog, ppath, map[string]interface{}{
            "Post": post,
            "Prev": nil,
            "Next": nil,
        })
        page.Title = post.Title
        if idx > 0 {
            pages[idx - 1].Content["Next"] = page
            page.Content["Prev"] = pages[idx - 1]
        }
        pages = append(pages, page)
    }

    return
}

func (idx *Index) Render(blog *Blog) (err error) {

    var errs = []error{}
    var conf = blog.Conf
    var fpath string
    var outf *os.File
    var flags int


    if _, err = os.Stat(conf.Dst); os.IsNotExist(err) {
        if err = os.MkdirAll(conf.Dst, 0755); err != nil {
            return
        }
    }

    //if tmpd, err = ioutil.TempDir(conf.Dst, "pub"); err != nil {
    //    return
    //}
    pages := idx.Pages(blog)
    postTpl := blog.Theme.Templates["post.html"]
    indexTpl := blog.Theme.Templates["index.html"]

    for _, page := range pages {
        fpath = path.Join(conf.Dst, page.Path)
        flags = os.O_CREATE|os.O_TRUNC|os.O_WRONLY

        if outf, err = os.OpenFile(fpath, flags, 0666); err != nil {
            fmt.Println("ERR", err)
            return
        }

        if err = postTpl.ExecuteTemplate(outf, "post.html", page); err != nil {
            errs = append(errs, err)
        }
        if err = outf.Close(); err != nil {
            errs = append(errs, err)
        }
    }

    fpath = path.Join(conf.Dst, "index.html")
    flags = os.O_CREATE|os.O_TRUNC|os.O_WRONLY

    if outf, err = os.OpenFile(fpath, flags, 0666); err != nil {
        fmt.Println("ERR", err)
        return
    }
    page := NewPage(blog, "index.html", map[string]interface{}{
        "Pages": pages,
    })
    page.Title = "Posts"
    if err = indexTpl.ExecuteTemplate(outf, "index.html", page); err != nil {
        errs = append(errs, err)
    }
    if err = outf.Close(); err != nil {
        errs = append(errs, err)
    }

    fmt.Println("ERRORS:", errs)

    return
}

func NewIndex(conf *Config) *Index {
    var err error
    var posts = []*Post{}

    walk := func (path string, finfo os.FileInfo, e error) (err error) {
        if e != nil {
            return
        }

        if path == conf.Src || finfo.IsDir() {
            return
        }

        posts = append(posts, NewPost(path))

        return
    }

    if err = filepath.Walk(conf.Src, walk); err != nil {
        return nil
    }

    return &Index{
        Posts: posts,
    }
}


func Run(config string) (err error) {
    conf := &Config{}
    buf := []byte{}
    root := ""

    if _, err = os.Stat(config); err != nil {
        return
    }

    if buf, err = ioutil.ReadFile(config); err != nil {
        return
    }

    if err = json.Unmarshal(buf, &conf); err != nil {
        return
    }

    if root, err = filepath.Abs(path.Dir(config)); err != nil {
        return
    }

    conf.Src = path.Join(root, conf.Src)
    conf.Theme = path.Join(root, conf.Theme)
    conf.Dst = path.Join(root, conf.Dst)
    fmt.Println(NewTheme(conf.Theme))

    idx := NewIndex(conf)
    idx.Render(NewBlog(conf))

    return
}



func main() {
    fmt.Println("mixxer")
    if err := Run("examples/one/config.json"); err != nil {
        fmt.Println("ERROR:", err)
    }
}

