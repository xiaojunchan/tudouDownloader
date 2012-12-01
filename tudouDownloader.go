package main

import (
    "net/http"
    "fmt"
    "io/ioutil"
    "flag"
    "encoding/json"
    "regexp"
    "strconv"
    "strings"
    "os"
    "io"
)

const (
    TUDOU_LIST  = "http://www.tudou.com/tva/srv/alist.action?ver=asins&a=" // return json
    TUDOU_VIDEO = "http://v2.tudou.com/v.action?retc=1&vn=02&hd=2&sid=10200&st=1&it=" // return xml
)

var (
    tudouUrl = flag.String("url", "", "tudou url")
    start    = flag.Int("start", 1, "start")
    end      = flag.Int("end", 1, "end")
)

type TudouList struct {
    Items    []TudouListItem
}

type TudouListItem struct {
    Kw      string
    Iid     int

    downUrl string
}

// get page content
func getPage(u string) []byte {
    resp, err := http.Get(u)
    if err != nil {
        panic(err)
    }

    body, err := ioutil.ReadAll(resp.Body)
    if err !=  nil {
        panic(err)
    }

    return body
}

// get download url
func getDownUrl(id string) string {
    id = strings.TrimSpace(id)

    fmt.Println("Getting download url, id:", id)

    downUrl     := TUDOU_VIDEO + id
    pageContent := getPage(downUrl)

    exp := regexp.MustCompile("<f.*>(.*)</f>")

    result := exp.FindStringSubmatch(string(pageContent))

    if  len(result) == 2 {
        return result[1]
    }

    return ""
}

// download file
func download(u, fn string) {
    fn = strings.TrimSpace(fn)

    // create folder
    os.Mkdir("output", 0774)

    // filter filename
    exp := regexp.MustCompile("[\\|/]")
    tmp := exp.ReplaceAll([]byte(fn), []byte("-"))
    fn  = string(tmp)

    resp, err := http.Get(u)
    if err != nil {
        panic(err)
    }

    fn += ".f4v"

    fmt.Println("Downloading..., filename:", fn)

    // create file
    file, err := os.OpenFile("output/" + fn, os.O_RDWR | os.O_APPEND | os.O_CREATE, 0664)
    if err != nil {
        panic(err)
    }
    defer file.Close()

    // downloading
    if _, err := io.Copy(file, resp.Body); err != nil {
        fmt.Println("Download Failed!")
    }

    fmt.Println("Download Success!")
}

// new album list
func newList(id string) *TudouList {
    id = strings.TrimSpace(id)

    listUrl     := TUDOU_LIST + id
    pageContent := getPage(listUrl)

    tudouList := new(TudouList)

    if err := json.Unmarshal(pageContent, tudouList); err != nil {
        panic(err)
    }

    return tudouList
}

// download album list
func (tl *TudouList) downList(start, end int) {
    if start < 0 {
        start = 1
    }

    if end - 1 > len(tl.Items) || end < start {
        end = len(tl.Items)
    }

    for start <= end {
        downUrl := getDownUrl(strconv.Itoa(tl.Items[start - 1].Iid))
        download(downUrl, tl.Items[start - 1].Kw)
        start++
    }
}

func printUsing() {
    fmt.Println("Using: ./tudou_downloader -url http://www.tudou.com/{albumplay, ?}/xxx/xxx (-start 1 -end 10)")
}

func main() {
    defer func() {
        if r := recover(); r != nil {
            fmt.Println(r)
        }
    }()

    flag.Parse()

    if *tudouUrl == "" {
        printUsing()
        os.Exit(0)
    }

    tudouPage := getPage(*tudouUrl)

    var exp   *regexp.Regexp
    var vtype string

    if strings.Contains(*tudouUrl, "albumplay") {
        if *start == 1 && *end == 1 {
            exp = regexp.MustCompile("iid:(.*)")
            vtype = "single"
        } else {
            exp   = regexp.MustCompile(",aid=(.*)")
            vtype = "album"
        }
    } else {
        exp = regexp.MustCompile("iid:(.*)")
        vtype = "single"
    }

    matchResult := exp.FindStringSubmatch(string(tudouPage))
    id := matchResult[1]

    switch vtype {
    case "album":
        downList := newList(id)
        downList.downList(*start, *end)
    case "single":
        downUrl := getDownUrl(id)
        download(downUrl, id)
    }
}
