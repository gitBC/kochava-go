package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"github.com/go-redis/redis"
)


func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	RedisServer := os.Getenv("REDIS_SERVER")
	RedisPort := os.Getenv("REDIS_PORT")

	client := redis.NewClient(&redis.Options{
		Addr:     RedisServer + ":" + RedisPort,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	/*
	Get a random value from a key on the stack
	 */
	fmt.Println(client.Get(client.RandomKey().Val()).Val()	)
}
