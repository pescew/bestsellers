package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

const (
	IMG_DIR     = "./img/"
	JSON_DIR    = "./json/"
	REWRITE_URL = ""
)

var LISTS = map[string]*Bestsellers{
	"combined-print-and-e-book-fiction":    nil,
	"combined-print-and-e-book-nonfiction": nil,
}

type response struct {
	Status       string  `json:"status"`
	Copyright    string  `json:"copyright"`
	NumResults   int     `json:"num_results"`
	LastModified string  `json:"last_modified"`
	Results      results `json:"results"`
}

type results struct {
	ListName                 string `json:"list_name"`
	ListNameEncoded          string `json:"list_name_encoded"`
	BestsellersDate          string `json:"bestsellers_date"`
	PublishedDate            string `json:"published_date"`
	PublishedDateDescription string `json:"published_date_description"`
	NextPublishedDate        string `json:"next_published_date"`
	PreviousPublishedDate    string `json:"previous_published_date"`
	DisplayName              string `json:"display_name"`
	NormalListEndsAt         int    `json:"normal_list_ends_at"`
	Updated                  string `json:"updated"`
	Books                    []book `json:"books"`
}

type book struct {
	Rank             int    `json:"rank"`
	RankLastWeek     int    `json:"rank_last_week"`
	WeeksOnList      int    `json:"weeks_on_list"`
	Asterisk         int    `json:"asterisk"`
	Dagger           int    `json:"dagger"`
	PrimaryISBN10    string `json:"primary_isbn10"`
	PrimaryISBN13    string `json:"primary_isbn13"`
	Publisher        string `json:"publisher"`
	Description      string `json:"description"`
	Price            string `json:"price"`
	Title            string `json:"title"`
	Author           string `json:"author"`
	Contributor      string `json:"contributor"`
	ContributorNote  string `json:"contributor_note"`
	BookImage        string `json:"book_image"`
	BookImageWidth   int    `json:"book_image_width"`
	BookImageHeight  int    `json:"book_image_height"`
	AmazonProductURL string `json:"amazon_product_url"`
	AgeGroup         string `json:"age_group"`
	BookReviewLink   string `json:"book_review_link"`
	ISBNs            []isbn `json:"isbns"`
}

type isbn struct {
	ISBN10 string `json:"isbn10"`
	ISBN13 string `json:"isbn13"`
}

type Bestsellers struct {
	LastModified string `json:"LastModified"`
	ListName     string `json:"ListName"`
	// ListNameEncoded          string        `json:"list_name_encoded"`
	// DisplayName              string        `json:"display_name"`
	// Updated                  string        `json:"updated"`
	BestsellersDate string `json:"ListDate"`
	// PublishedDate            string        `json:"published_date"`
	// PublishedDateDescription string        `json:"published_date_description"`
	Books []BookCompact `json:"Books"`
}

type BookCompact struct {
	Rank int `json:"Rank"`
	// RankLastWeek     int    `json:"rank_last_week"`
	// WeeksOnList      int    `json:"weeks_on_list"`
	// Asterisk         int    `json:"asterisk"`
	// Dagger           int    `json:"dagger"`
	// PrimaryISBN10 string `json:"ISBN10"`
	PrimaryISBN13 string `json:"ISBN13"`
	// Publisher     string `json:"Publisher"`
	// Description   string `json:"Description"`
	Title  string `json:"Title"`
	Author string `json:"Author"`
	// Contributor     string `json:"Contributor"`
	// ContributorNote string `json:"ContributorNote"`
	// BookImage       string `json:"Image"`
	BookImageWidth  int `json:"ImageWidth"`
	BookImageHeight int `json:"ImageHeight"`
	// AmazonProductURL string `json:"amazon_product_url"`
	// AgeGroup         string `json:"age_group"`
	// BookReviewLink string `json:"ReviewLink"`
	// ISBNs            []isbn `json:"isbns"`
}

func (b *book) compact() *BookCompact {
	return &BookCompact{
		Rank: b.Rank,
		// PrimaryISBN10: b.PrimaryISBN10,
		PrimaryISBN13: b.PrimaryISBN13,
		// Publisher:     b.Publisher,
		// Description:     b.Description,
		Title:  b.Title,
		Author: b.Author,
		// Contributor:     b.Contributor,
		// ContributorNote: b.ContributorNote,
		// BookImage:       b.BookImage,
		BookImageWidth:  b.BookImageWidth,
		BookImageHeight: b.BookImageHeight,
	}
}

func (r *response) bestsellers() *Bestsellers {
	bs := Bestsellers{
		LastModified:    r.LastModified,
		ListName:        r.Results.ListName,
		BestsellersDate: r.Results.BestsellersDate,
		Books:           []BookCompact{},
	}

	for _, bk := range r.Results.Books {
		bs.Books = append(bs.Books, *bk.compact())
	}

	return &bs
}

func main() {
	fmt.Println("Starting...")

	imageURLs := make(map[string]string)
	for listName, _ := range LISTS {
		resp, err := readJSONFromFile(fmt.Sprintf("./%s.json", listName))
		if err != nil {
			panic(err)
		}
		bs := resp.bestsellers()
		LISTS[listName] = bs

		for _, bk := range resp.Results.Books {
			imageURLs[bk.PrimaryISBN13] = bk.BookImage
		}
	}

	err := clearDirectory(JSON_DIR)
	if err != nil {
		panic(err)
	}

	for listName, list := range LISTS {
		err = WriteJsonFile(listName, *list)
		if err != nil {
			panic(err)
		}
	}

	err = clearDirectory(IMG_DIR)
	if err != nil {
		panic(err)
	}

	for isbn, imgURL := range imageURLs {
		fileName, err := downloadImage(imgURL, isbn)
		if err != nil {
			fmt.Printf("Error downloading image: %s\n", imgURL)
			fmt.Println(err)
			continue
		}
		fmt.Printf("Downloaded image: %s\n", fileName)
		time.Sleep(1 * time.Second)
	}

}

func readJSONFromFile(filename string) (*response, error) {
	var resp response
	fileData, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(fileData, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func readJSONFromUrl(url string) (*response, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	respByte := buf.Bytes()
	var obj response
	err = json.Unmarshal(respByte, &obj)
	if err != nil {
		return nil, err
	}

	return &obj, nil
}

func clearDirectory(dir string) error {
	err := os.RemoveAll(dir)
	if err != nil {
		return err
	}
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	return nil
}

func downloadImage(imgURL string, isbn string) (string, error) {
	urlPath, err := url.Parse(imgURL)
	if err != nil {
		return "", err
	}
	imgName := path.Base(urlPath.Path)
	extension := path.Ext(imgName)

	outFileName := fmt.Sprintf("%s%s", isbn, extension)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(imgURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		return "", fmt.Errorf("response content-type is not an image: %s", contentType)
	}

	out, err := os.Create(fmt.Sprintf("%s%s", IMG_DIR, outFileName))
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return outFileName, nil
}

func WriteJsonFile(name string, list Bestsellers) error {
	buffer := new(bytes.Buffer)
	encoder := json.NewEncoder(buffer)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(list)
	if err != nil {
		return err
	}

	fileName := fmt.Sprintf("%s%s.json", JSON_DIR, name)

	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(buffer.Bytes())
	if err != nil {
		return err
	}

	return nil
}
