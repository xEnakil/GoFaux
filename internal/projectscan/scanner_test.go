package projectscan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanDetectsExternalClientsAndInternalRoutes(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "src/main/java/demo/PeopleClient.java", `
package demo;

@FeignClient(name = "peopleClient", url = "https://partner.example.com")
public interface PeopleClient {
  @GetMapping("/v1/people")
  PeopleResponse listPeople();
}
`)
	writeFile(t, root, "src/main/java/demo/PeopleController.java", `
package demo;

@RestController
@RequestMapping("/api")
public class PeopleController {
  @PostMapping("/people")
  PeopleResponse createPeople(@RequestBody PeopleRequest request) { return null; }
}
`)
	writeFile(t, root, "frontend/api.ts", `
import axios from "axios";
export const getPayments = () => axios.get("https://payments.example.com/v2/payments");
router.get("/internal/status", handler);
`)

	preview, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}

	assertIntegration(t, preview, "spring-feign", "client", true, "GET", "/v1/people")
	assertIntegration(t, preview, "spring-controller", "server", false, "POST", "/api/people")
	assertIntegration(t, preview, "axios", "client", true, "GET", "/v2/payments")
	assertIntegration(t, preview, "express-route", "server", false, "GET", "/internal/status")
}

func TestScanDetectsGoAndPythonHTTPClients(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "client.go", `
package demo

import "net/http"

func call() {
  _, _ = http.Get("https://inventory.example.com/items")
  req, _ := http.NewRequest("POST", "https://orders.example.com/orders", nil)
  _ = req
}
`)
	writeFile(t, root, "client.py", `
import requests
requests.delete("https://users.example.com/v1/users/42")
`)

	preview, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}

	assertIntegration(t, preview, "go-http-client", "client", true, "GET", "/items")
	assertIntegration(t, preview, "go-http-client", "client", true, "POST", "/orders")
	assertIntegration(t, preview, "python-http-client", "client", true, "DELETE", "/v1/users/42")
}

func writeFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertIntegration(t *testing.T, preview Preview, kind, direction string, external bool, method, endpoint string) {
	t.Helper()
	for _, integration := range preview.Integrations {
		if integration.Kind == kind &&
			integration.Direction == direction &&
			integration.External == external &&
			integration.Method == method &&
			integration.Endpoint == endpoint {
			return
		}
	}
	t.Fatalf("missing integration kind=%s direction=%s external=%v method=%s endpoint=%s in %#v", kind, direction, external, method, endpoint, preview.Integrations)
}
