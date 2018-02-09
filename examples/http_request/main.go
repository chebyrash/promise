package main

import (
	"fmt"
	"github.com/chebyrash/promise"
	"io/ioutil"
	"net/http"
	"sync"
)

func main() {
	var wg = &sync.WaitGroup{}
	wg.Add(1)

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
		wg.Done()
	})

	requestPromise.Catch(func(error error) {
		fmt.Println(error.Error())
		wg.Done()
	})

	wg.Wait()
}
