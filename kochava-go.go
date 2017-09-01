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


type Statistics struct {
	delivery_attempts, response_code int
	response_time int64
	response_body, original_redis_key string
}

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
//	QueueLocation = "http://koc.app"


	statistics := Statistics{0, 0,0,"",Key}

	//Create http client to work on Redis Items
	HttpClient := &http.Client{}

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

			statistics.delivery_attempts++
			statistics.response_body = RequestBody
			statistics.response_code = resp.StatusCode
			statistics.response_time = time.Now().UnixNano() - deliveryTime

			updateStatistics()

			// delete successfully delivered key
			client.Del(Key)

			//we delivered, bone out
			break

		} else {

			statistics.delivery_attempts++
			fmt.Println(err)

			if i == RedisDeliveryAttempts - 1 {

				statistics.response_time = time.Now().UnixNano() - deliveryTime
				updateStatistics()
			}
			fmt.Println("try again")

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

}
