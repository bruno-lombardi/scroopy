package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
)

type Contact struct {
	Name         string
	Address      string
	Neighborhood string
	ZipCode      string
	Phone        string
}

func NewContact(name string, address string, neighborhood string, zipcode string, phone string) *Contact {
	return &Contact{
		Name:         name,
		Address:      address,
		Neighborhood: neighborhood,
		ZipCode:      zipcode,
		Phone:        phone,
	}
}

func removeExceedingWhiteSpace(val string) string {
	r, _ := regexp.Compile(`\s\s+`)
	res := r.ReplaceAllString(val, "")
	return res
}

func extractZipCode(val *string) string {
	r, _ := regexp.Compile(`\d{5}-\d{3}`)
	zipCode := r.FindString(*val)
	*val = r.ReplaceAllString(*val, "")
	return zipCode
}

func extractNeighborhood(val string) string {
	zipCodeRegex, _ := regexp.Compile(`\d{5}`)
	values := strings.Split(val, "-")
	values[1] = zipCodeRegex.ReplaceAllString(values[1], "")
	values[1] = strings.TrimLeft(removeExceedingWhiteSpace(values[1]), " ")
	return values[1]
}

func extractNumbers(val string) string {
	r, _ := regexp.Compile(`\d+`)
	phone := r.FindString(val)
	return phone
}

func sanitizeAddress(val string) string {
	values := strings.Split(val, "-")
	return strings.Trim(values[0], " ")
}

func main() {
	// builder := strings.Builder{}
	contacts := []*Contact{}

	c := colly.NewCollector(
		colly.AllowedDomains("listatelefonica.tk"),
	)
	var page int64

	c.OnHTML(".card-body", func(h *colly.HTMLElement) {
		if len(h.DOM.Children().Nodes) <= 15 {
			return
		}

		url := h.Request.URL.String()
		page, _ = strconv.ParseInt(h.Request.URL.Query().Get("page"), 10, 0)
		fmt.Printf("scraping %v\n", url)

		h.ForEach("a", func(i int, h *colly.HTMLElement) {
			link := h.Attr("href")
			if !strings.Contains(link, "/sp/detalhes") {
				// builder.WriteString(fmt.Sprintf("%v\n", link))
				h.Request.Visit(link)
			}
		})

		if page%5 == 0 || page >= 457 {
			write("contacts.csv", contacts)
			contacts = contacts[:0]
		}
	})

	c.OnHTML(".card.mb-3", func(h *colly.HTMLElement) {
		name := h.ChildText("h1.card-title")
		if name == "" {
			return
		}
		addressText := removeExceedingWhiteSpace(h.ChildText("address"))
		phoneElement := h.DOM.Find("a[href=\"javascript:void(0);\"]")
		phoneNumbersAttr, _ := phoneElement.Attr("ng-click")

		zipCode := extractZipCode(&addressText)
		neighborhood := extractNeighborhood(h.ChildText("address"))
		phone := extractNumbers(phoneNumbersAttr)
		address := sanitizeAddress(addressText)

		contact := NewContact(name, address, neighborhood, zipCode, phone)
		contacts = append(contacts, contact)
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("error in request: %v", err)
	})

	c.OnHTML("#app > div > div.row > div.col-12.col-md-9 > div:nth-child(3) > div.card-footer > ul > li > a[rel=\"next\"]", func(h *colly.HTMLElement) {
		href := h.Attr("href")
		h.Request.Visit(href)
	})

	c.Visit("https://listatelefonica.tk/cidade/jundiai/sp?page=1")
}

func write(fileName string, contacts []*Contact) error {
	var data [][]string
	if _, err := os.Stat(fileName); err != nil {
		// File does not exist, first row is headers
		row := []string{"Nome", "Endereço", "Bairro", "CEP", "Telefone"}
		data = append(data, row)
	}

	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	w := csv.NewWriter(file)
	defer w.Flush()

	for _, contact := range contacts {
		row := []string{contact.Name, contact.Address, contact.Neighborhood, contact.ZipCode, contact.Phone}
		data = append(data, row)
	}
	w.WriteAll(data)
	return file.Close()
}
