package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/chebyrash/promise"
)

func parseJSON(data []byte) *promise.Promise {
	return promise.New(func(resolve func(promise.Any), reject func(error)) {
		var body = make(map[string]string)

		err := json.Unmarshal(data, &body)
		if err != nil {
			reject(err)
		}

		resolve(body)
	})
}

func main() {
	var requestPromise = promise.New(func(resolve func(promise.Any), reject func(error)) {
		resp, err := http.Get("https://httpbin.org/ip")
		defer resp.Body.Close()
		if err != nil {
			reject(err)
			return
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			reject(err)
			return
		}
		resolve(body)
	})

	// Parse JSON body in async manner
	parsed, err := requestPromise.
		Then(func(data promise.Any) promise.Any {
			// This can be a promise, it will automatically flatten
			return parseJSON(data.([]byte))
		}).Await()

	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		return
	}

	origin := parsed.(map[string]string)["origin"]
	fmt.Printf("Origin: %s\n", origin)
}
