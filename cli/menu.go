package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"GoFaux/api"
	"GoFaux/server"
)

func RunMenu() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\n🌟 Mock API Server")
		fmt.Println("1 Add new API")
		fmt.Println("2 View existing APIs")
		fmt.Println("3 Remove API")
		fmt.Println("4 Start Server")
		fmt.Println("5 Exit")
		fmt.Println("------------------------------")
		fmt.Print("👉 Enter your choice: ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			addApi(reader)
		case "2":
			api.ViewAPIs()
		case "3":
			removeApi(reader)
		case "4":
			go server.StartServer()
		case "5":
			fmt.Println("👋 Exiting... Goodbye!")
			os.Exit(0)
		default:
			fmt.Println("❌ Invalid choice. Try again.")
		}
	}
}

func addApi(reader *bufio.Reader) {

	fmt.Print("🔹 Enter API method (GET/POST/PUT/DELETE): ")
	method, _ := reader.ReadString('\n')
	method = strings.TrimSpace(strings.ToUpper(method))

	fmt.Print("🔹 Enter API endpoint (e.g., /users): ")
	endpoint, _ := reader.ReadString('\n')
	endpoint = strings.TrimSpace(endpoint)

	fmt.Print("🔹 Enter JSON response: ")
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(response)

	if !api.IsValidJSON(response) {
		fmt.Println("❌ Invalid JSON format. Try again.")
		return
	}

	api.AddApi(method, endpoint, response)
}

func removeApi(reader *bufio.Reader) {
	api.ViewAPIs()

	fmt.Print("\n❌ Enter API number to remove: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	index, err := strconv.Atoi(input)
	if err != nil {
		fmt.Println("❌ Invalid number.")
		return
	}

	if api.RemoveAPI(index - 1) {
		fmt.Println("✅ API removed successfully.")
	} else {
		fmt.Println("❌ Invalid API number.")
	}
}