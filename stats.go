package main

import "fmt"

func Stats(n int) {
	var cursor uint64
	pattern := "*" + statsKey + "*"
	for {
		keys, newCursor, err := Redis.Scan(ctx, cursor, pattern, 10).Result()
		if err != nil {
			fmt.Println(fmt.Errorf("error scanning Redis keys: %w", err))
			return
		}

		for _, key := range keys {
			result, err := Redis.Get(ctx, key).Result()
			if err != nil {
				fmt.Printf("error retrieving key %s: %v\n", key, err)
				continue
			}
			fmt.Printf("key: %s, result: %s\n", key, result)
		}

		cursor = newCursor
		if cursor == 0 {
			break
		}
	}
	Exit()
}
