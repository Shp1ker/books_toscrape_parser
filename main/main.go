package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/ivangurin/cbrf-go"
	"github.com/xuri/excelize/v2"
)

func main() {
	referal := "https://books.toscrape.com/"

	var titles, hrf []string
	var rating []int
	var price []float64

	currencyRate, err := getGBPRate("GBP")
	if err != nil {
		log.Fatalf("Ошибка при получении курса: %v\n", err)
	}
	key := true
	for key == true {
		res, err := http.Get(referal)
		if err != nil {
			log.Fatal(err)
		}
		defer res.Body.Close()

		if res.StatusCode != 200 {
			log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
		}

		doc, err := goquery.NewDocumentFromReader(res.Body)
		if err != nil {
			log.Fatal(err)
		}
		class_name := "section article"

		doc.Find(class_name).Each(func(i int, item *goquery.Selection) {
			tag := item.Find("p")
			tmp, exists := tag.Attr("class")
			if !exists {
				rating = append(rating, 0)
			} else {
				switch tmp {
				case "star-rating One":
					rating = append(rating, 1)
				case "star-rating Two":
					rating = append(rating, 2)
				case "star-rating Three":
					rating = append(rating, 3)
				case "star-rating Four":
					rating = append(rating, 4)
				case "star-rating Five":
					rating = append(rating, 5)
				}
			}
			tag = item.Find("h3 a")
			tmp, exists = tag.Attr("href")
			if !exists {
				hrf = append(hrf, "-")
			} else {
				tmp = strings.Replace(tmp, "catalogue/", "", -1)
				hrf = append(hrf, "https://books.toscrape.com/catalogue/"+tmp)

				title, exists := tag.Attr("title")
				if exists {
					titles = append(titles, title)
				} else {
					titles = append(titles, "-")
				}
			}
			tag = item.Find("div p")
			num64 := getPrice(tag) * currencyRate
			num64 = math.Round(num64)
			price = append(price, num64)
		})
		next_page := doc.Find("li.next a").First()

		href, exists := next_page.Attr("href")
		if !exists {
			key = false
			break
		}
		href = strings.Replace(href, "catalogue/", "", -1)

		referal = "https://books.toscrape.com/catalogue/" + href
	}
	f := excelize.NewFile()
	index, _ := f.NewSheet("Books")
	f.SetActiveSheet(index)

	f.SetCellValue("Books", "A1", "Название")
	f.SetCellValue("Books", "B1", "Оценка")
	f.SetCellValue("Books", "C1", "Цена")
	f.SetCellValue("Books", "D1", "Ссылка")

	row := 2

	for i := range titles {
		f.SetCellValue("Books", fmt.Sprintf("A%d", row), titles[i])
		f.SetCellValue("Books", fmt.Sprintf("B%d", row), rating[i])
		f.SetCellValue("Books", fmt.Sprintf("C%d", row), price[i])
		f.SetCellValue("Books", fmt.Sprintf("D%d", row), hrf[i])
		row++
	}

	if err := f.SaveAs("books.xlsx"); err != nil {
		log.Fatal(err)
	}
}

func getGBPRate(valute string) (float64, error) {
	ctx := context.Background()
	now := time.Now().UTC()

	exchangeRate, err := cbrf.GetExchangeRate(ctx, valute, now)
	if err != nil {
		return 0, err
	}
	return exchangeRate, nil
}

func getPrice(sel *goquery.Selection) float64 {
	raw := sel.Text()

	priceStr := strings.ReplaceAll(raw, "£", "")
	priceStr = strings.TrimSpace(priceStr)
	priceStr = strings.SplitN(priceStr, "\n", 2)[0]

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		log.Printf("Ошибка парсинга цены '%s': %v", raw, err)
		return 0
	}
	return price
}
