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
	"math"
)

var RedisServer, RedisPort string
var RedisDeliveryAttempts int
var client *redis.Client

/**
	For JSON.Marshall to actually work on a struct, not only must you use the json tagging, but also "export"
	the variables for usage by naming them with a capital letter?
 */
type Statistics struct {
	Delivery_attempts   int `json:"delivery_attempts"`
	Response_code       int `json:"response_code"`
	Response_body       string `json:"response_body"`
	Response_time_delta string `json:"response_time_delta"` //we're sending more data than needed calc on logging server
	Delivery_time_delta string `json:"delivery_time_delta"`
	Response_datetime string `json:"response_datetime"` //we're sending more data than needed calc on logging server
	Delivery_datetime string `json:"delivery_datetime"`
	Original_redis_key  string `json:"original_redis_key"`
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
	requestJSON := client.LPop("queue:requests").Val()

	byt := []byte(requestJSON)

	//exit application if there is no length
	if len(byt) <= 0 {
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
	QueueTime := dat["original_request_time"].(string)

	//Teporary override of domain
	QueueLocation = "http://koc.app/"

	statistics = Statistics{Original_redis_key: QueueTime}

	//Store the current time so that we can count total response time.
	DeliveryStartTime := time.Now()

	for i := 0; i < RedisDeliveryAttempts; i++ {

		req, _ := http.NewRequest(QueueMethod, QueueLocation, nil)
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

			ResponseReceivedTime := time.Now()

			//Get redis key, convert to a float, then make a new time object which we can subtract from current time
			//TODO: ensure times are set the same on ingestion and delivery servers
			original_request_time, err := strconv.ParseFloat(QueueTime, 64)
			if err != nil {
			}

			sec, dec := math.Modf(original_request_time);
			original_request_time_time := time.Unix(int64(sec), int64(dec*(1e9)))

			//Subtract initial request time from now to get the total amount of time it took for us to deliver
			statistics.Delivery_time_delta = durationToMicroString(ResponseReceivedTime.Sub(original_request_time_time))
			statistics.Delivery_datetime = timeToMicroString(DeliveryStartTime)

			//subtract one time object from another, ouput difference in seconds, format to string, 6 digits
			statistics.Response_time_delta = durationToMicroString(ResponseReceivedTime.Sub(DeliveryStartTime))
			statistics.Response_datetime = timeToMicroString(ResponseReceivedTime)

			statistics.Delivery_attempts++
			statistics.Response_body = RequestBody
			statistics.Response_code = resp.StatusCode

			updateStatistics()

			//we delivered, bone out
			break

		} else {
			ResponseReceivedTime := time.Now()

			statistics.Delivery_attempts++
			fmt.Println(err)

			if statistics.Delivery_attempts == RedisDeliveryAttempts {

				//stupid magic number to get microseconds to store in php
				statistics.Response_time_delta = durationToMicroString(ResponseReceivedTime.Sub(DeliveryStartTime))

				statistics.Delivery_datetime = timeToMicroString(DeliveryStartTime)
				statistics.Response_datetime = timeToMicroString(ResponseReceivedTime)
				updateStatistics()

			} else {

				fmt.Println("try again")

			}

		}

	}

}


func durationToMicroString(timeToConvert time.Duration) string {
	return strconv.FormatFloat(timeToConvert.Seconds(), 'f', 6, 64)
}
func timeToMicroString(timeToConvert time.Time) string {
	return strconv.FormatInt(timeToConvert.Unix(),10) + "." + strconv.Itoa(timeToConvert.Nanosecond())
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
func updateStatistics() {
	fmt.Println(statistics)

	//Should be able to serialize these, getting empty object back
	sendem, err := json.Marshal(statistics)
	if err != nil {
		fmt.Println(err)
		return
	}

	response, posterr := HttpClient.Post(os.Getenv("DETAILS_API_LOCATION"), "application/json", bytes.NewReader(sendem))

	if posterr != nil {
		fmt.Println(response)
	}

}
