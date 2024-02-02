package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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
	specificLocations := []int64{86719856975, 85644706127, 75901960467, 87891673423}
	app := goshopify.App{
		ApiKey:    os.Getenv("ApiKey"),
		ApiSecret: os.Getenv("ApiSecret"),
	}
	client := goshopify.NewClient(app, os.Getenv("ShopName"), os.Getenv("AccessToken"))

	file, err := os.Create("products.csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"SKU", "EAN", "Title", "Description", "Tags", "Price", "Soggeto iv o no", "Quantità", "Immagini"}
	writer.Write(header)

	var pagination *goshopify.Pagination

	for {
		options := &goshopify.ProductListOptions{
			ListOptions: goshopify.ListOptions{Limit: 250},
		}

		products, newPagination, err := client.Product.ListWithPagination(options)
		if err != nil {
			panic(err)
		}
		pagination = newPagination
		for _, product := range products {
			for _, variant := range product.Variants {
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
				time.Sleep(250 * time.Millisecond)
			}
		}
		if pagination.NextPageOptions == nil {
			break
		}

		options.Page = pagination.NextPageOptions.Page
	}
	fmt.Println("done")
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
