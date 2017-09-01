package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"github.com/go-redis/redis"
	"reflect"
	"encoding/json"
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
	Key := client.RandomKey().Val()

	Va := client.Get(Key).Val()

	fmt.Println(reflect.TypeOf(Va))

	byt := []byte(Va)
	//We need to provide a variable where the JSON package can put the decoded data. This map[string]interface{} will hold a map of strings to arbitrary data types.
	var dat map[string]interface{}
	if err := json.Unmarshal(byt, &dat); err != nil {
		panic(err)
	}
	fmt.Println(dat)

	//fmt.Println(Value)
}
