package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
)

const (
	batchSize = 10000
	filename  = "./data/top10milliondomains.csv"
)

func LoadJob(n int) {
	cleanUpStats()
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("could not open file: %w", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// Skip the header row
	if _, err := reader.Read(); err != nil {
		log.Printf("could not read header: %v", err)
		return
	}
	batch := make([]string, 0, batchSize)

	for {
		// Read each record from the CSV file
		rawRecord, err := reader.Read()
		if err != nil {
			if err.Error() == "EOF" {
				// Add the last batch if there are any remaining domains
				if len(batch) > 0 {
					err = SAdd(key+jobQueue, batch)
					if err != nil {
						log.Printf("could not add last batch to Redis: %w", err)
						return
					}
				}
				break
			}
			log.Printf("could not read CSV data: %w", err)
			return
		}

		// Ensure the record has at least two columns
		if len(rawRecord) < 2 {
			continue
		}

		// Add the second column (domain) to the batch
		batch = append(batch, rawRecord[1])

		// If the batch reaches the specified size, add it to Redis
		if len(batch) >= batchSize {
			err = SAdd(key+jobQueue, batch)
			if err != nil {
				log.Printf("could not add batch to Redis: %w", err)
				return
			}
			// Reset the batch
			batch = batch[:0]
		}
	}

	log.Println("LoadJob is done!")
	Exit()
}

func cleanUpStats() {
	var cursor uint64
	pattern := "*" + statsKey + "*"
	for {
		keys, newCursor, err := Redis.Scan(ctx, cursor, pattern, 10).Result()
		if err != nil {
			fmt.Println(fmt.Errorf("error scanning Redis keys: %w", err))
			return
		}
		fmt.Printf("number of keys %d", len(keys))
		for _, key := range keys {
			result, err := Redis.Del(ctx, key).Result()
			if err != nil {
				fmt.Printf("error retrieving key %s: %v\n", key, err)
				continue
			}
			fmt.Printf("key: %s, result: %d", key, result)
		}

		cursor = newCursor
		if cursor == 0 {
			break
		}
	}
}
