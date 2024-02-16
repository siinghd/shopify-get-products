package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	// "strings"
	"time"

	goshopify "github.com/bold-commerce/go-shopify/v3"
	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("Starting")
	err := godotenv.Load(filepath.Join("./", ".env"))
	if err != nil {
		log.Fatal("Error loading .env file", err)
	}
	go getProducts()

	fmt.Println("Starting server on port 4069")
	http.HandleFunc("/download", downloadHandler)
	log.Fatal(http.ListenAndServe(":4069", nil))
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Disposition", "attachment; filename=products.csv")
	w.Header().Set("Content-Type", "text/csv")
	http.ServeFile(w, r, "final_products.csv")
}

func getProducts() {
	app := goshopify.App{
		ApiKey:    os.Getenv("ApiKey"),
		ApiSecret: os.Getenv("ApiSecret"),
	}
	client := goshopify.NewClient(app, os.Getenv("ShopName"), os.Getenv("AccessToken"))
	// ticker := time.NewTicker(200 * time.Millisecond) // Set up a ticker that ticks every 500ms (2 calls per second)
	// defer ticker.Stop()
	productsLenghtSum := 0
	specificLocations := []int64{86719856975, 85644706127, 75901960467, 87891673423}

	for {

		file, err := os.Create("products.csv")
		if err != nil {
			panic(err)
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		writer.Comma = '|'
		defer writer.Flush()

		header := []string{"SKU", "EAN", "Title", "Description", "Tags", "Price", "Soggeto iv o no", "Quantità", "Immagini"}
		writer.Write(header)

		options := &goshopify.ProductListOptions{
			ListOptions: goshopify.ListOptions{Limit: 250},
		}

		for {

			waitIfNeeded(client)
			// <-ticker.C
			products, newPagination, err := client.Product.ListWithPagination(options)
			fmt.Println(err)
			if err != nil {
				panic(err)
			}

			// paginationJSON, err := json.MarshalIndent(newPagination, "", "  ")
			// if err != nil {
			// 	log.Fatalf("Error marshalling pagination: %v", err)
			// }
			// fmt.Println("Pagination info:", string(paginationJSON))

			productsLenghtSum += len(products)
			fmt.Println(productsLenghtSum)

			for _, product := range products {
				for _, variant := range product.Variants {

					waitIfNeeded(client)
					quantity, err := getQuantityForInventoryItem(client, variant.InventoryItemId, specificLocations)
					if err != nil {
						log.Fatalf("Error fetching quantity: %v", err)
					}

					var sku, ean string
					if len(product.Variants) > 0 {
						sku = variant.Sku
						ean = variant.Barcode
					} else {
						continue
					}

					images := getProductImages(product.Images)
					var taxable string

					if variant.Taxable {
						taxable = "soggeto iv"
					} else {
						taxable = "no"
					}
					record := []string{
						sku,
						ean,
						product.Title,
						product.BodyHTML,
						product.Tags,
						variant.Price.String(),
						taxable,
						fmt.Sprintf("%d", quantity),
						strings.Join(images, ", "),
					}
					writer.Write(record)
				}
			}
			if newPagination.NextPageOptions == nil {
				break
			}
			options.ListOptions.PageInfo = newPagination.NextPageOptions.PageInfo
		}
		copyFile("./products.csv", "./final_products.csv")
		fmt.Println("done")
		time.Sleep(60 * time.Minute)

	}
}

func waitIfNeeded(client *goshopify.Client) {
	rateInfo := client.RateLimits
	if rateInfo.BucketSize > 0 {
		requestsLeft := rateInfo.BucketSize - rateInfo.RequestCount
		if requestsLeft < 5 {
			timeToReset := 3 * time.Second // Assuming a reset every second
			time.Sleep(timeToReset)
		}
	}
}

func getQuantityForInventoryItem(client *goshopify.Client, inventoryItemID int64, locationIDs []int64) (int, error) {
	totalQuantity := 0

	inventoryLevels, err := client.InventoryLevel.List(goshopify.InventoryLevelListOptions{
		LocationIds:      locationIDs,
		InventoryItemIds: []int64{inventoryItemID},
	})
	if err != nil {
		return 0, err
	}

	for _, inventory := range inventoryLevels {
		totalQuantity += inventory.Available
	}

	return totalQuantity, nil
}

func getProductImages(images []goshopify.Image) []string {
	var urls []string
	for _, img := range images {
		urls = append(urls, img.Src)
	}
	return urls
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	err = os.Chmod(dst, sourceInfo.Mode())
	if err != nil {
		return err
	}

	return nil
}
