package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
	uuid "github.com/satori/go.uuid"
)

/*
	 	(1, 'Все товары', NULL),
	    (2, 'Ноутбуки и планшеты', 1),
	    (3, 'Планшеты', 2),
	    (4, 'Ноутбуки', 2),
	    (5, 'Бытовая техника', 1),
	    (6, 'Холодильники', 5),
	    (7, 'Стиральные машины', 5),
	    (8, 'Пылесосы', 5),
		(9, 'Мебель', 1),
		(91, 'Стулья', 9),
		(92, 'Рабочии столы', 9),
		(93, 'Диваны', 9),
		(94, 'Кресла', 9),
*/
var catMap map[int]string = map[int]string{
	6:  "holodilniki",
	7:  "stiralnye-mashiny",
	8:  "pylesosy",
	91: "stulya",
	92: "rabochie-stoly",
	93: "divany",
	94: "kresla",
}

func GetParse(category_id int) []string {
	rows := make([]string, 20)

	options := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36"),
	)

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), options...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	var html string
	err := chromedp.Run(ctx,
		// visit the target page
		chromedp.Navigate("https://megamarket.ru/catalog/"+catMap[category_id]+"/"),
		// wait for the page to load
		chromedp.Sleep(1000*time.Millisecond),
		// extract the raw HTML from the page
		chromedp.ActionFunc(func(ctx context.Context) error {
			// select the root node on the page
			rootNode, err := dom.GetDocument().Do(ctx)
			if err != nil {
				return err
			}
			html, err = dom.GetOuterHTML().WithNodeID(rootNode.NodeID).Do(ctx)
			return err
		}),
	)
	if err != nil {
		log.Fatal("Error while performing the automation logic:", err)
	}
	//if err := os.WriteFile("file.html", []byte(html), 0666); err != nil {
	//	log.Fatal(err)
	//}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader([]byte(html)))
	if err != nil {
		log.Fatalln(err)
	}

	/*content, err := os.ReadFile("./html/" + catMap[category_id] + ".html")
	if err != nil {
		log.Fatalln(err)
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(content))
	if err != nil {
		log.Fatalln(err)
	}*/

	doc.Find("div.catalog-item-mobile").Each(func(i int, s *goquery.Selection) {
		getNum := func(r rune) rune {
			if !unicode.IsNumber(r) {
				return -1
			}
			return r
		}
		photo := s.Find("div.item-image").Find("a").Find("div.catalog-item-photo").Find("img")
		imgsrc, _ := photo.Attr("src")
		name, _ := photo.Attr("alt")

		id := uuid.NewV4()

		info := s.Find("div.item-block").Find("div.item-info")
		price_str := info.Find("div.catalog-item-mobile__prices-container").Find("div.item-money").Find("div.item-price").Text()
		review_str := info.Find("div.inner").Find("div.item-review").Find("a").Find("div.item-review-wrapper").Find("div.review-amount").Text()
		review_ := strings.Fields(review_str)
		if len(review_[0]) < 1 {
			log.Fatalln("review_ Fields empty")
		}
		review, err := strconv.Atoi(review_[0])
		if err != nil {
			log.Fatalln(err)
		}
		rating := review % 100

		price := strings.Map(getNum, price_str)

		photoName := strconv.Itoa(category_id) + "-" + id.String() + filepath.Ext(imgsrc)
		WritePhoto(imgsrc, photoName, category_id)

		row := fmt.Sprintf("('%s', '%s', %s, '%s', '%s', 4.%d, %d),",
			id.String(), name, price, photoName, "Самый лучший среди товаров на рынке "+name, rating, category_id)
		rows = append(rows, row)
		fmt.Print(".")
	})

	rows[len(rows)-1] = replaceLastRune(rows[len(rows)-1], ';')
	return rows
}

func Filler(category_id int) {
	f, err := os.Create("./sql/filler_" + catMap[category_id] + ".sql")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := os.MkdirAll("./photos/"+catMap[category_id]+"/", os.ModePerm); err != nil {
		log.Fatal(err)
	}

	lines := GetParse(category_id)

	preCMD := `INSERT INTO product (id, name, price, imgsrc, description, rating, category_id)
	VALUES`
	f.WriteString(preCMD)
	for _, line := range lines {
		_, err := f.WriteString(line + "\n")
		if err != nil {
			log.Fatal(err)
		}
	}
}

func WritePhoto(url string, fileName string, category_id int) {
	response, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	defer response.Body.Close()
	name := "./photos/" + catMap[category_id] + "/" + fileName
	file, err := os.Create(name)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		log.Fatal(err)
	}
	t := time.NewTimer(1 * time.Second)
	<-t.C
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalln("invalid len arg")
	}
	category_id, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}
	cat, ok := catMap[category_id]
	if !ok {
		log.Fatalln("invalid category_id")
	}

	fmt.Println("	Start parsing <", cat, ">")
	Filler(category_id)
}

func replaceLastRune(s string, new rune) string {
	old, size := utf8.DecodeLastRuneInString(s)
	if old == utf8.RuneError && size <= 1 {
		return s
	}
	return s[:len(s)-size] + string(new)
}
