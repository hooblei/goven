
package goven

import (
    "fmt"
    "os"
    "path"
    "time"
    "sort"
    "io/ioutil"
    "encoding/json"
    "path/filepath"
    "text/template"
)


type SiteConfig struct {
    SourceDir string `json:"sourceDir"`
    PubDir string `json:"pubDir"`
    ThemeDir string `json:"themeDir"`
    BaseUrl string `json:"baseUrl"`
    Title string `json:"title"`
    Owner string `json:"owner"`
    DateFormat string `json:"dateFormat"`
    Root string
    BasePath string
}

func NewSiteConfig(fpath string) *SiteConfig {
    var err error

    config := &SiteConfig{
        Title: "A goven baked site",
        BaseUrl: "http://www.example.com/",
        Owner: "you <you@example.com>",
        ThemeDir: "./theme",
        SourceDir: "./posts",
        PubDir: "./public",
    }
    buf := []byte{}
    root := ""

    if _, err = os.Stat(fpath); err != nil {
        panic(err)
    }

    if buf, err = ioutil.ReadFile(fpath); err != nil {
        panic(err)
    }

    if root, err = filepath.Abs(path.Dir(fpath)); err != nil {
        panic(err)
    }

    if err = json.Unmarshal(buf, &config); err != nil {
        panic(err)
    }

    config.SourceDir = path.Join(root, config.SourceDir)
    config.PubDir = path.Join(root, config.PubDir)
    config.ThemeDir = path.Join(root, config.ThemeDir)

    return config
}


type Site struct {
    Config *SiteConfig
    Theme *Theme
}

func (s *Site) Url(p string) string {
    return path.Join(s.Config.BaseUrl, p)
}

func NewSite(config *SiteConfig) *Site {
    return &Site{
        Config: config,
        Theme: NewTheme(config.ThemeDir),
    }
}


type PageTpl struct {
    Name string
    Template *template.Template
}

func (t *PageTpl) Exec(outf *os.File, page *Page) error {
    return t.Template.ExecuteTemplate(outf, t.Name, page);
}


type Page struct {
    Site *Site
    Cdt time.Time
    Path string
    Title string
    Template *PageTpl
    Content map[string]interface{}
}

func (p *Page) Url() string {
    return p.Site.Url(p.Path)
}

func (p *Page) Save() (err error) {
    var outf *os.File

    fpath := path.Join(p.Site.Config.PubDir, p.Path)
    flags := os.O_CREATE | os.O_TRUNC | os.O_WRONLY

    if outf, err = os.OpenFile(fpath, flags, 0666); err != nil {
        return
    }

    if err = p.Template.Exec(outf, p); err != nil {
        return
    }

    if err = outf.Close(); err != nil {
        return
    }

    return
}

type PageContent map[string]interface{}

func NewPage(site *Site, path string, tpl *PageTpl, content PageContent) *Page {
    return &Page{
        Site: site,
        Cdt: time.Now(),
        Path: path,
        Template: tpl,
        Content: content,
    }
}


type Pages []*Page

func (p Pages) Save() (err error) {
    var page *Page

    for _, page = range p {
        if err = page.Save(); err != nil {
            return
        }
    }

    return
}


type Blog struct {
    *Site
    Posts Posts
}

func NewBlog(config *SiteConfig) *Blog {
    return &Blog{
        Site: NewSite(config),
        Posts: NewPosts(config.SourceDir),
    }
}

func (b *Blog) PostsPages() Pages {
    pages := Pages{}
    sort.Sort(PostsByCdt{b.Posts})
    tpl := &PageTpl{
        Name: "post.html",
        Template: b.Site.Theme.Templates["post.html"],
    }
    for idx, post := range b.Posts {
        ppath := fmt.Sprintf("%s.html", post.Slug())
        page := NewPage(b.Site, ppath, tpl, PageContent{
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

    return pages
}

func (b *Blog) PostsIndex() Pages {
    sort.Sort(PostsByCdt{b.Posts})
    pages := Pages{}
    tpl := &PageTpl{
        Name: "index.html",
        Template: b.Site.Theme.Templates["index.html"],
    }

    return append(pages, NewPage(b.Site, "index.html", tpl, PageContent{
        "Pages": b.PostsPages(),
    }))
}

func (b *Blog) Build() (err error) {
    if err = b.PostsPages().Save(); err != nil {
        return
    }
    if err = b.PostsIndex().Save(); err != nil {
        return
    }
    return
}


type Theme struct {
    Path string
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
       Path: tpath,
       Templates: tmap,
   }
}

