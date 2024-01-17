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
	"github.com/chai2010/webp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
	uuid "github.com/satori/go.uuid"
)

/*
INSERT INTO category
VALUES

	(1, 'Все товары', NULL),

	(2, 'Электроника', 1),
	(21, 'Планшеты', 2),
	(22, 'Ноутбуки', 2),
	(23, 'Мониторы', 2),
	(24, 'Наушники', 2),

	(3, 'Бытовая техника', 1),
	(31, 'Холодильники', 3),
	(32, 'Стиральные машины', 3),
	(33, 'Пылесосы', 3),

	(4, 'Музыкальные инструменты', 1),
	(41, 'Губные гармошки', 4),
	(42, 'Гитары', 4),
	(43, 'Барабаны', 4),
	(44, 'Клавишные', 4),
	(45, 'Смычковые музыкальные инструменты', 4),
	(46, 'Духовые музыкальные инструменты', 4),
	(47, 'Виниловые пластинки', 4),

	(5, 'Спорт и активный отдых', 1),
	(51, 'Велосипеды', 5),
	(52, 'Горные лыжи', 5),
	(53, 'Сноуборды', 5),
	(54, 'Самокаты', 5),
	(55, 'Веревки альпинистские', 5),
	(56, 'Дартс', 5),

	(6, 'Красота и уход', 1),
	(61, 'Уход за лицом', 6),
	(62, 'Средства по уходу за волосами', 6),
	(63, 'Косметика для макияжа лица', 6),
	(64, 'Макияж глаз', 6),

	(7, 'Ювелирные изделия', 1),
	(71, 'Кольца', 7),
	(72, 'Серьги', 7),
	(73, 'Браслеты', 7),
	(74, 'Цепочки', 7),
	(75, 'Колье', 7),

	(8, 'Новогодние товары', 1),
	(81, 'Елки искусственные', 8),
	(82, 'Живые елки', 8),
	(83, 'Аксессуары для елок', 8),
	(84, 'Елочные украшения', 8),
	(85, 'Новогодние гирлянды светодиодные', 8),

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
	21: "planshety",
	22: "noutbuki",
	23: "monitory",
	24: "naushniki",

	31: "holodilniki",
	32: "stiralnye-mashiny",
	33: "pylesosy",

	41: "gubnye-garmoshki-i-aksessuary",
	42: "gitary-i-gitarnoe-oborudovanie",
	43: "udarnye",
	44: "klavishnye",
	45: "smychkovye-muzykalnye-instrumenty",
	46: "duhovye-muzykalnye-instrumenty",
	47: "vinilovye-plastinki",

	51: "velosipedy",
	52: "gornye-lyzhi",
	53: "snoubordy",
	54: "samokaty",
	55: "verevki-alpinistskie",
	56: "misheni-dlya-dartsa",

	61: "uhod-za-licom",
	62: "sredstva-po-uhodu-za-volosami",
	63: "makiyazh-lica",
	64: "makiyazh-glaz",

	71: "kolca",
	72: "sergi",
	73: "braslety",
	74: "cepochki",
	75: "kole",

	81: "iskusstvennye-elki",
	82: "zhivye-elki",
	83: "aksessuary-dlya-elok",
	84: "elochnye-igrushki-i-ukrasheniya",
	85: "novogodnie-girlyandy-svetodiodnye",

	91: "stulya",
	92: "rabochie-stoly",
	93: "divany",
	94: "kresla",

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

		photoName := strconv.Itoa(category_id) + "-" + id.String() + ".webp"
		err := WritePhotoWebp(imgsrc, photoName, category_id)
		if err != nil {
			return
		}
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

	for k := range catMap {
		preCMD := "INSERT INTO product (id, name, price, imgsrc, description, rating, category_id) VALUES\n"
		lines := GetParse(k)
		f.WriteString(preCMD)
		for _, line := range lines {
			_, err := f.WriteString(line + "\n")
			if err != nil {
				log.Fatal(err)
			}
		}
		fmt.Println("category id done: ", k)
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

func WritePhotoWebp(url string, fileName string, category_id int) error {
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	img, err := jpeg.Decode(response.Body)
	if err != nil {
		return err
	}

	name := "./images/" + fileName
	output, err := os.Create(name)
	if err != nil {
		return err
	}
	defer output.Close()

	if err := webp.Encode(output, img, &webp.Options{Lossless: false, Quality: 70}); err != nil {
		return err
	}

	t := time.NewTimer(10 * time.Millisecond)
	<-t.C
	return nil
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
	return strings.Replace(strings.Replace(s, "/", " ", -1), "'", "", -1)

}
