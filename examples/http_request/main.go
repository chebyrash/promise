package main

import (
	"encoding/json"
	"fmt"
	"github.com/chebyrash/promise"
	"io/ioutil"
	"net/http"
)

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
			var body = make(map[string]string)

			json.Unmarshal(data.([]byte), &body)

			return body
		}).
		// Work with parsed body
		Then(func(data interface{}) interface{} {
			var body = data.(map[string]string)

			fmt.Println("Origin:", body["origin"])

			return nil
		}).
		Catch(func(error error) error {
			fmt.Println(error.Error())
			return nil
		})

	requestPromise.Await()
}
