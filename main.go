package main

import (
	"bytes"
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
	uuid "github.com/satori/go.uuid"
)

var catMap map[int]string = map[int]string{
	8: "pylesosy",
}

func GetParse(category_id int) []string {
	rows := make([]string, 20)

	content, err := os.ReadFile("./" + catMap[category_id] + ".html")
	if err != nil {
		log.Fatalln(err)
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(content))
	if err != nil {
		log.Fatalln(err)
	}

	doc.Find("div.catalog-item-mobile").Each(func(i int, s *goquery.Selection) {
		photo := s.Find("div.item-image").Find("a").Find("div.catalog-item-photo").Find("img")
		imgsrc, _ := photo.Attr("src")
		name, _ := photo.Attr("alt")

		id := uuid.NewV4()

		info := s.Find("div.item-block").Find("div.item-info")
		price_str := info.Find("div.catalog-item-mobile__prices-container").Find("div.item-money").Find("div.item-price").Text()
		review_str := info.Find("div.inner").Find("div.item-review").Find("a").Find("div.item-review-wrapper").Find("div.review-amount").Text()
		review, err := strconv.Atoi(strings.TrimSpace(review_str))
		if err != nil {
			log.Fatalln(err)
		}
		rating := review % 100

		getNum := func(r rune) rune {
			if !unicode.IsNumber(r) {
				return -1
			}
			return r
		}
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
	// remember to close the file
	defer f.Close()
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
	Filler(8)
}

func replaceLastRune(s string, new rune) string {
	old, size := utf8.DecodeLastRuneInString(s)
	if old == utf8.RuneError && size <= 1 {
		return s
	}
	return s[:len(s)-size] + string(new)
}
