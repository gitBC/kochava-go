package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"github.com/go-redis/redis"
	"encoding/json"
	"net/http"
	"bytes"
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

	byt := []byte(Va)

	/*
	We need to provide a variable where the JSON package can put the decoded data. This map[string]interface{} will hold a map of strings to arbitrary data types.
	 */
	var dat map[string]interface{}
	if err := json.Unmarshal(byt, &dat); err != nil {
		panic(err)
	}

	QueueMethod := dat["method"]
	QueueLocation := dat["location"]



	//Create http client to work on Redis Items
	HttpClient := &http.Client{}

	req, _ := http.NewRequest(QueueMethod.(string), QueueLocation.(string), nil)
	req.Header.Add("Accept", "application/json")
	resp, err := HttpClient.Do(req)

	defer resp.Body.Close()


	//Create a buffer to read the response body
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)

	//Create and print string from response body
	RequestBody := buf.String()
	fmt.Println(RequestBody)


}
