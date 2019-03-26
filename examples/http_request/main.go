package main

import (
	"encoding/json"
	"fmt"
	"github.com/chebyrash/promise"
	"io/ioutil"
	"net/http"
)

func parseJSON(data []byte) *promise.Promise {
	return promise.New(func(resolve func(interface{}), reject func(error)) {
		var body = make(map[string]string)

		err := json.Unmarshal(data, &body)
		if err != nil {
			reject(err)
		}

		resolve(body)
	})
}

func main() {
	var requestPromise = promise.New(func(resolve func(interface{}), reject func(error)) {
		var url = "https://httpbin.org/ip"

		resp, err := http.Get(url)
		defer resp.Body.Close()

		if err != nil {
			reject(err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			reject(err)
		}

		resolve(body)
	})

	requestPromise.
		// Parse JSON body
		Then(func(data interface{}) interface{} {
			// This can be a promise, it will automatically flatten
			return parseJSON(data.([]byte))
		}).
		// Work with parsed body
		Then(func(data interface{}) interface{} {
			var body = data.(map[string]string)

			fmt.Println("Origin:", body["origin"])

			return body
		}).
		Catch(func(error error) error {
			fmt.Println(error.Error())
			return nil
		})

	// Your resolved values can be extracted from the Promise
	// But you are encouraged to handle them in .Then and .Catch
	value, err := requestPromise.Await()

	if err != nil {
		fmt.Println("Error: " + err.Error())
	}

	origin := value.(map[string]string)["origin"]
	fmt.Println("Origin: " + origin)
}
