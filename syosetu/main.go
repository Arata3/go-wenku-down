package main

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/astaxie/bat/httplib"
	"github.com/gosuri/uilive"
)

var baseURL string
var nID string
var writer *uilive.Writer
var fID *os.File

func init() {

	nID = "n2267be"
	baseURL = "http://ncode.syosetu.com/"

	writer = uilive.New()

	nTxtFile := path.Join(nID + ".md")

	os.Mkdir(nID, 0744)
	var err error
	fID, err = os.Create(nTxtFile)

	if err != nil {
		os.Exit(0)
	}

}

func main() {

	writer.Start()

	indexURL := baseURL + nID
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

	indexDom, err := goquery.NewDocumentFromResponse(reqp)
	// goquery.NewDocument(indexURL)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// title
	title := indexDom.Find(".novel_title").Text()
	info := indexDom.Find("#novel_ex").Text()
	fmt.Printf("[书名] %s\n%s\n", title, info)
	fID.WriteString("#" + title)
	fID.WriteString("\n" + info + "\n\n")

	tdsDoms := indexDom.Find(".index_box .subtitle")
	getContent(tdsDoms)

	fID.Close()

	writer.Stop()
	fmt.Fprintf(writer, "SUCCESS!\n")

}

// 内容获取
func getContent(d *goquery.Selection) {
	tmpCount := d.Length()
	fmt.Printf("[基本路径] %s [%d]\n", baseURL, tmpCount)
	// 定义开始每卷章节号

	d.Each(func(i int, c *goquery.Selection) {

		aLink := c.Find("a")
		tmpLink, have := aLink.Attr("href")
		title := aLink.Text()
		if have {
			tmpfileName := strconv.Itoa(i) + ".md"
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

			allDom := tmpDom.Find("#novel_honbun")

			content := allDom.Text()
			// fmt.Println(content)

			fID.WriteString(fmt.Sprintf("\n## %s\n", title))
			fID.WriteString(content)

			tmpfile, _ := os.Create(nID + "/" + tmpfileName)
			tmpfile.WriteString(fmt.Sprintf("## %s\n", title))
			tmpfile.WriteString(content)
			tmpfile.Close()

			fmt.Fprintf(writer.Bypass(), "[OVER] %s\n", title)

		}

		time.Sleep(time.Millisecond * 25)
	})
}
