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
	"strconv"
	"time"
)

var RedisServer, RedisPort string
var RedisDeliveryAttempts int
var client *redis.Client


/**
	For JSON.Marshall to actually work on a struct, not only must you use the json tagging, but also "export"
	the variables for usage by naming them with a capital letter?
 */
type Statistics struct {

	Delivery_attempts int `json:"delivery_attempts"`
	Response_code int `json:"response_code"`
	Response_time int64 `json:"response_time"`
	Response_body string `json:"response_body"`
	Original_redis_key string `json:"original_redis_key"`
}

var statistics Statistics

//Create http client to work on Redis Items
var HttpClient = &http.Client{}

func main() {

	//load godotenv, set env variables.
	bootstrap()

	//Connect to redis, set the client interface to and instance of redis.Client
	connectToRedis()

	/*
	Get a random value from a key on the stack
	 */
	Key := client.RandomKey().Val()
	Va := client.Get(Key).Val()

	byt := []byte(Va)

	if len(byt) == 0 {
		os.Exit(0)
	}

	/*
	 * We need to provide a variable where the JSON package can put the decoded data. This map[string]interface{} will
	 * hold a map of strings to arbitrary data types.
	 */
	var dat map[string]interface{}
	if err := json.Unmarshal(byt, &dat); err != nil {
		panic(err)
	}

	//Set a few variables we will use when delivering the Redis item
	QueueMethod := dat["method"].(string)
	QueueLocation := dat["location"].(string)

	//Teporary override of domain
	QueueLocation = "http://koc.app/"


	//statistics = Statistics{0, 0,0,"",Key}
	statistics = Statistics{Original_redis_key:Key}

	//Store the current nano time so that we can count total response time.
	deliveryTime := time.Now().UnixNano();


	fmt.Println(deliveryTime);

	for i := 0 ; i < RedisDeliveryAttempts; i++ {

		req, _ := http.NewRequest(QueueMethod, QueueLocation,nil)
		req.Header.Add("Accept", "application/json")
		resp, err := HttpClient.Do(req)


		//Check for request error
		if nil == err {

			fmt.Println(resp)

			//HttpClient.Do automatically uses the provided transport to close the body on non-nil response
			//defer resp.Body.Close()


			//Create a buffer to read the response body
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)

			//Create and print string from response body
			RequestBody := buf.String()
			fmt.Println(RequestBody)

			fmt.Println(resp.StatusCode)

			statistics.Delivery_attempts++
			statistics.Response_body = RequestBody
			statistics.Response_code = resp.StatusCode

			//stupid magic number to get microseconds to store in php
			statistics.Response_time = (time.Now().UnixNano() - deliveryTime) / 1000

			updateStatistics()

			// delete successfully delivered key
			client.Del(Key)

			//we delivered, bone out
			break

		} else {

			statistics.Delivery_attempts++
			fmt.Println(err)

			if statistics.Delivery_attempts == RedisDeliveryAttempts {

				//stupid magic number to get microseconds to store in php
				statistics.Response_time = (time.Now().UnixNano() - deliveryTime) / 1000
				updateStatistics()
				client.Del(Key)

			} else {

				fmt.Println("try again")

			}

		}

	}

}


/**
Load Environment variables
 */
func bootstrap() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	RedisServer = os.Getenv("REDIS_SERVER")
	RedisPort = os.Getenv("REDIS_PORT")
	RedisDeliveryAttempts, err = strconv.Atoi(os.Getenv("REDIS_DELIVERY_ATTEMPTS"))
}

/**
Creates a redis client for the function
 */
func connectToRedis() {
	client = redis.NewClient(&redis.Options{
		Addr:     RedisServer + ":" + RedisPort,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}


/**
	Sends updated statistics to PHP endpoint to track success / failure of delivery:wq
 */
func updateStatistics(){
	fmt.Println(statistics)

	//Should be able to serialize these, getting empty object back
	sendem, err := json.Marshal( statistics)
	if err != nil {
		fmt.Println(err)
		return
	}

	response,posterr := HttpClient.Post(os.Getenv("DETAILS_API_LOCATION"), "application/json", bytes.NewReader(sendem) )

	if posterr != nil {
		fmt.Println(response)
	}

}
