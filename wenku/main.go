package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"time"
	// "container/list"
	"net/http"
	_ "net/http/pprof"
	"regexp"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/astaxie/bat/httplib"
	"github.com/gosuri/uilive"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

var baseURL string
var nID string
var writer *uilive.Writer
var fID, fIndex *os.File
var wg sync.WaitGroup

func init() {

	if len(os.Args) == 1 {
		fmt.Println("输入书号")
		os.Exit(0)
	}

	nID = os.Args[1]
	if ok, _ := regexp.MatchString(`\d{1,6}`, nID); !ok {
		fmt.Println("输入错误")
		os.Exit(0)
	}

	numID, err := strconv.Atoi(nID)
	if err != nil {
		os.Exit(0)
	}

	var nKu = "0"

	if numID > 999 {
		nKu = "1"
	}

	if numID > 1999 {
		nKu = "2"
	}

	baseURL = "http://www.wenku8.com/novel/" + nKu + "/" + nID + "/"

	writer = uilive.New()

	nTxtFile := path.Join(nID, nID+".md")
	nIndexFile := path.Join(nID, "index.md")

	os.Mkdir(nID, 0744)
	fID, err = os.Create(nTxtFile)
	fIndex, err = os.Create(nIndexFile)
	if err != nil {
		os.Exit(0)
	}
}

func main() {

	//这里实现了远程获取pprof数据的接口
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	writer.Start()
	fmt.Printf("[书号] %s\n", nID)
	indexURL := baseURL + "index.htm"

	reqp, err := httplib.Get(indexURL).SetTimeout(time.Second*10, time.Second*10).Response()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	if reqp.StatusCode == 404 {
		fmt.Println(indexURL)
		fmt.Println(404)
		os.Exit(0)
	}

	// indexDom, err := goquery.NewDocumentFromReader(reqp.Body)
	indexDom, err := goquery.NewDocumentFromResponse(reqp)
	// goquery.NewDocument(indexURL)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// title
	title := gbk2utf(indexDom.Find("#title").Text())
	info := gbk2utf(indexDom.Find("#info").Text())
	fmt.Printf("[书名] %s\n%s\n", title, info)

	fIndex.WriteString(fmt.Sprintf("# %s\n%s\n\n", title, info))

	// vcss
	tdsDoms := indexDom.Find("table tr td")
	getContent(tdsDoms)

	// listAlink := indexDom.Find(".ccss a")
	// getContent(listAlink, f)

	fID.Close()
	fIndex.Close()

	// 等待清零
	wg.Wait()
	writer.Stop()
	fmt.Fprintf(writer, "SUCCESS!\n")
}

// 内容获取
func getContent(d *goquery.Selection) {
	fmt.Printf("[基本路径] %s\n", baseURL)
	tmpCount := d.Length()
	// 定义开始每卷章节号
	var tipc = 0
	var tipv = 0
	d.Each(func(i int, c *goquery.Selection) {
		// 卷头
		if c.HasClass("vcss") {
			tipv++
			t1 := gbk2utf(c.Text())
			fmt.Fprintf(writer, "[#] %s", t1)
			fID.WriteString("# " + t1)
			fIndex.WriteString("\n* " + t1)
			tipc = 0
		}

		if c.HasClass("ccss") {
			aLink := c.Find("a")
			tmpLink, have := aLink.Attr("href")
			title := gbk2utf(aLink.Text())
			if have {
				tipc++
				tmpfileName := strconv.Itoa(tipv) + "_" + strconv.Itoa(tipc) + ".md"
				fmt.Fprintf(writer, "[下载文章][%d/%d] %s\n", i, tmpCount, title)
				fmt.Fprintf(writer, "[下载路径] %s%s\n", baseURL, tmpLink)
				tmpResponse, err := httplib.Get(baseURL+tmpLink).SetTimeout(time.Second*10, time.Second*10).Response()

				if err != nil {
					fmt.Println(err)
					os.Exit(0)
				}

				// tmpDom, err := goquery.NewDocument(baseURL + tmpLink)
				// tmpDom, err := goquery.NewDocumentFromReader(tmpResponse.Body)
				tmpDom, err := goquery.NewDocumentFromResponse(tmpResponse)
				if err != nil {
					fmt.Println(err)
				}

				allDom := tmpDom.Find("#content")

				content := gbk2utf(allDom.Text())
				// fmt.Println(content)

				fID.WriteString(fmt.Sprintf("\n## %s\n", title))
				fID.WriteString(content)
				fIndex.WriteString(fmt.Sprintf("\n\t* [%s](%s)", title, tmpfileName))

				tmpfile, _ := os.Create(nID + "/" + tmpfileName)
				tmpfile.WriteString(fmt.Sprintf("## %s\n", title))
				tmpfile.WriteString(content)
				tmpfile.Close()

				listImgs := allDom.Find("img")
				if listImgs.Length() > 0 {
					// +1
					wg.Add(1)
					go getImage(listImgs, tmpfileName)
				}

				fmt.Fprintf(writer.Bypass(), "[OVER] %s\n", title)

			}
		}
		time.Sleep(time.Millisecond * 25)
	})
}

func getImage(d *goquery.Selection, fname string) {
	var tmpCount = d.Length()
	f, _ := os.Open(nID + "/" + fname)
	defer f.Close()
	// -1
	defer wg.Done()
	d.Each(func(i int, c *goquery.Selection) {
		tmpLink, have := c.Attr("src")
		if have {
			req := httplib.Get(tmpLink).SetTimeout(time.Second*10, time.Second*10)
			resp, err := req.Response()
			if err != nil {
				log.Println(err)
				return
			}
			status := resp.StatusCode
			// header := resp.Header
			fmt.Fprintf(writer, "[下载图片][%d/%d]%s [%d]\n", i, tmpCount, tmpLink, status)

			tmpPath := path.Join("imgs", path.Base(tmpLink))
			f.WriteString(fmt.Sprintf("\n* ![%s](%s)", tmpPath, tmpPath))

			if status == 200 {
				data, _ := req.Bytes()
				writeImg(path.Join(nID, tmpPath), data)
				return
			}
		}
	})
	f.Close()
}

func writeImg(url string, b []byte) error {
	dir, _ := path.Split(url)

	err := os.MkdirAll(dir, 0755)
	if err != nil {
		fmt.Println("--wirteDir-->", err)
	}

	fImg, err := os.Create(url)
	_, err = fImg.Write(b)

	// err = p.req.ToFile(pf)
	if err != nil {
		fmt.Println("--wirteFile-->", err)
	}
	fImg.Close()

	return err
}

// utf2gbk 是 转为 GBK 编码
func utf2gbk(src string) (dst string) {
	data, err := ioutil.ReadAll(transform.NewReader(bytes.NewReader([]byte(src)), simplifiedchinese.GBK.NewEncoder()))
	if err == nil {
		dst = string(data)
	}
	return
}

// gbk2utf 是 GBK 转 utf8编码
func gbk2utf(src string) (dst string) {
	data, err := ioutil.ReadAll(transform.NewReader(bytes.NewReader([]byte(src)), simplifiedchinese.GBK.NewDecoder()))
	if err == nil {
		dst = string(data)
	}
	return
}
