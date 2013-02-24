
package main

import (
    "fmt"
    "os"
    "path"
    "flag"
    "io"
    "strings"
    "text/template"
    "goven"
)

var usageTpl = `goven is a tool for baking a static site out of some text files

Usage:

    goven command [arguments]

Available commands:
    {{ range . }}
        {{.Name | printf "%-11s"}} {{.Short}}{{end}}

Use "goven help [command]" for more information about a command.
`

var helpTpl = `Usage: goven {{.Usage}}

{{.Descr}}
`

func exists(path string) (bool, error) {
    _, err := os.Stat(path)
    if err == nil { return true, nil }
    if os.IsNotExist(err) { return false, nil }
    return false, err
}

func isdir(path string) bool {
    var fi os.FileInfo
    var err error

    if fi, err = os.Stat(path); err != nil {
        if os.IsNotExist(err) {
            return false
        } else {
            panic(err)
        }
    }

    return fi.Mode().IsDir()
}

type Cmd struct {
    Usage string
    Short string
    Descr string
    Flags flag.FlagSet
    Run func(cmd *Cmd, args []string)
    X func()
}

func (c *Cmd) Name() string {
    name := c.Usage
    i := strings.Index(name, " ")
    if i >= 0 {
        name = name[:i]
    }

    return name
}

func (c *Cmd) PrintUsage() {
    fmt.Fprintf(os.Stderr, "usage: %s\n\n", c.Usage)
    if len(c.Descr) > 0 {
        fmt.Fprintf(os.Stderr, "%s\n", strings.TrimSpace(c.Descr))
    }
    os.Exit(2)
}



var bake = &Cmd{
    Usage: "bake [sites]",
    Short: "Bake one or more goven sites",
    Descr: "TODO descr",
    Run: func (cmd *Cmd, args []string) {
        if len(args) < 1 {
            args = append(args, ".")
        }

        for _, conf := range args {
            if isdir(conf) {
                conf = path.Join(conf, "goven.json");
            }
            // XXX: Error and exit on missing config?
            fmt.Printf("baking %q ... ", path.Dir(conf))
            b := goven.NewBlog(goven.NewSiteConfig(conf))
            fmt.Println("done")
            b.Build()
        }
    },
}


var help = &Cmd{
    Usage: "help command",
    Short: "Print Help contents for given command and exit",
    Run: func (cmd *Cmd, args []string) {
        if len(args) == 0 {
            usage()
            return
        }

        if len(args) != 1 {
            cmd.PrintUsage()
            os.Exit(2)
            return
        }

        arg := args[0]
        for _, cmd := range commands {
            if cmd.Name() == arg {
                tpl(os.Stdout, helpTpl, cmd)
                return
            }
        }

        fmt.Fprintf(os.Stderr, "Unknown goven help topic %q\n", arg)
        os.Exit(2)
    },
}

var commands []*Cmd

func init() {
    commands = append(commands,
        bake,
        help,
    )
}

func tpl(w io.Writer, t string, c interface{}) {
    tpl := template.New("top")
    template.Must(tpl.Parse(t))
    if err := tpl.Execute(w, c); err != nil {
        panic(err)
    }
}

func usage() {
    tpl(os.Stderr, usageTpl, commands)
    os.Exit(2)
}

func main () {
    flag.Parse()
    args := flag.Args()
    if len(args) < 1 {
        usage()
    }

    for _, cmd := range commands {
        if cmd.Name() == args[0] && cmd.Run != nil {
            cmd.Flags.Parse(args[1:])
            args = cmd.Flags.Args()
            cmd.Run(cmd, args)
            return
        }
    }

    fmt.Fprintf(os.Stderr, "goven: unknown command %q\nRun 'goven help' for usage and available commands.\n", args[0])
    os.Exit(2)
}

