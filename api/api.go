package api

import (
	"fmt"
)

type API struct {
	Method string
	Endpoint string
	Response string
	Requests int
}

func AddApi(method, endpoint, response string) {
	AddToStore(endpoint, method, response)
	fmt.Printf("✅ API '%s %s' added successfully!\n", method, endpoint)
}

func ViewAPIs() {
	apis := GetAllAPIs()
	fmt.Print("------------------------------")
	fmt.Println("\n📜 Mocked APIs:")

	if len(apis) == 0 {
		fmt.Println("❌ No APIs added yet.")
		fmt.Println("------------------------------")
		return
	}

	for i, api := range apis {
		fmt.Printf("[%d] %s %s - Requests:\n", i+1, api.Method, api.Endpoint, api.Requests)
	}

	fmt.Println("\n------------------------------")
}

func RemoveAPI(index int) bool {
	return DeleteFromStore(index)
}