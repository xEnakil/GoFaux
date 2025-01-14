package api

import (
	"sync"
)

var (
	apiStore []API
	mutex sync.Mutex
)

func AddToStore(endpoint, method, response string) {
	mutex.Lock()
	defer mutex.Unlock()
	apiStore = append(apiStore, API{Method: method, Endpoint: endpoint, Response: response, Requests: 0})
}

func GetAllAPIs() []API {
	mutex.Lock()
	defer mutex.Unlock()
	return apiStore
}

func DeleteFromStore(index int) bool {
	mutex.Lock()
	defer mutex.Unlock()
	if index < 0 || index >= len(apiStore) {
		return false
	}
	apiStore = append(apiStore[:index], apiStore[index+1:]...)
	return true
}

func IncrementRequestCount(endpoint string) {
	mutex.Lock()
	defer mutex.Unlock()
	for i, api:= range apiStore {
		if api.Endpoint == endpoint {
			apiStore[i].Requests++
		}
	}
}