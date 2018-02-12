package main

import (
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

		resolve(string(body))
	})

	requestPromise.Then(func(data interface{}) {
		fmt.Println(data)
	})

	requestPromise.Catch(func(error error) {
		fmt.Println(error.Error())
	})

	requestPromise.Await()
}
