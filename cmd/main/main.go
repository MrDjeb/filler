package main

import (
	"bytes"
	"context"
	"fmt"
	"image/jpeg"
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
	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
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
		(92, 'Рабочие столы', 9),
		(93, 'Диваны', 9),
		(94, 'Кресла', 9),
		(10, 'Канцелярия', 1),
		(101, 'Тетради', 10),
		(102, 'Письменные принадлежности', 10),
		(103, 'Пеналы', 10),
		(104, 'Клей', 10),
		(11, 'Товары для геймеров', 1),
		(111, 'Nintendo', 11),
		(112, 'Xbox', 11),
		(113, 'PlayStation', 11);
*/
var catMap map[int]string = map[int]string{
	6:   "holodilniki",
	7:   "stiralnye-mashiny",
	8:   "pylesosy",
	91:  "stulya",
	92:  "rabochie-stoly",
	93:  "divany",
	94:  "kresla",
	101: "tetradi",
	102: "pismennye-prinadlezhnosti",
	103: "penaly",
	104: "kley-455773",
	111: "nintendo",
	112: "xbox",
	113: "playstation",
}

func GetParse(category_id int) []string {
	rows := make([]string, 20)
	uniq := make(map[string]int)

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
	fmt.Println("	success pars ", "https://megamarket.ru/catalog/"+catMap[category_id]+"/")
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

		if _, ok := uniq[name]; ok {
			return
		}
		uniq[name] = 1

		id := uuid.NewV4()

		info := s.Find("div.item-block").Find("div.item-info")
		price_str := info.Find("div.catalog-item-mobile__prices-container").Find("div.item-money").Find("div.item-price").Text()
		review_str := info.Find("div.inner").Find("div.item-review").Find("a").Find("div.item-review-wrapper").Find("div.review-amount").Text()
		review_ := strings.Fields(review_str)
		var review int
		if len(review_) < 1 {
			review = 95
			//log.Println("review_ Fields empty", review_str)
		} else {
			review, err = strconv.Atoi(review_[0])
			if err != nil {
				log.Fatalln(err)
			}
		}

		rating := review % 100

		price := strings.Map(getNum, price_str)

		photoName := strconv.Itoa(category_id) + "-" + id.String()
		WritePhotoWebp(imgsrc, photoName, category_id)
		cleanName := sanitize(name)
		row := fmt.Sprintf("('%s', '%s', %s, '%s', '%s', 4.%d, %d),",
			id.String(), cleanName, price, photoName, "Самый лучший среди товаров на рынке "+cleanName, rating, category_id)
		rows = append(rows, row)
		fmt.Print(".")
	})
	fmt.Println()
	rows[len(rows)-1] = replaceLastRune(rows[len(rows)-1], ';')
	return rows
}

func Fill() {
	if err := os.MkdirAll("./one_sql/", os.ModePerm); err != nil {
		log.Fatal(err)
	}
	f, err := os.Create("./one_sql/9_filler.sql")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := os.MkdirAll("./images/", os.ModePerm); err != nil {
		log.Fatal(err)
	}

	for k, _ := range catMap {
		preCMD := "INSERT INTO product (id, name, price, imgsrc, description, rating, category_id) VALUES\n"
		f.WriteString(preCMD)
		lines := GetParse(k)
		for _, line := range lines {
			_, err := f.WriteString(line + "\n")
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func Filler(category_id int) {
	f, err := os.Create("./sql/9_filler_" + catMap[category_id] + ".sql")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := os.MkdirAll("./images/"+catMap[category_id]+"/", os.ModePerm); err != nil {
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
	name := "./images/" + fileName + filepath.Ext(url)
	file, err := os.Create(name)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		log.Fatal(err)
	}
	t := time.NewTimer(50 * time.Millisecond)
	<-t.C
}

func WritePhotoWebp(url string, fileName string, category_id int) {
	response, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	img, err := jpeg.Decode(response.Body)
	if err != nil {
		log.Fatalln(err)
	}

	name := "./images/" + fileName + filepath.Ext(url)
	output, err := os.Create(name + ".webp")
	if err != nil {
		log.Fatal(err)
	}
	defer output.Close()

	options, err := encoder.NewLossyEncoderOptions(encoder.PresetDefault, 75)
	if err != nil {
		log.Fatalln(err)
	}

	if err := webp.Encode(output, img, options); err != nil {
		log.Fatalln(err)
	}

	t := time.NewTimer(10 * time.Millisecond)
	<-t.C
}

func main() {
	/*if len(os.Args) != 2 {
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
	Filler(category_id)*/
	Fill()
}

func replaceLastRune(s string, new rune) string {
	old, size := utf8.DecodeLastRuneInString(s)
	if old == utf8.RuneError && size <= 1 {
		return s
	}
	return s[:len(s)-size] + string(new)
}

func sanitize(s string) string {
	return strings.Replace(s, "/", " ", -1)

}
