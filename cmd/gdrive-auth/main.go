package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

func main() {
	credentialsFile := flag.String("credentials", "oauth_client_secret.json", "Path to OAuth2 credentials.json file")
	tokenFile := flag.String("token", "gdrive_token.json", "Path to save token file (default: project base directory)")
	flag.Parse()

	b, err := os.ReadFile(*credentialsFile)
	if err != nil {
		log.Fatalf("Unable to read credentials file: %v", err)
	}

	config, err := google.ConfigFromJSON(b, drive.DriveFileScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	tok := getTokenWithFallback(config)
	saveToken(*tokenFile, tok)
	fmt.Printf("Token saved to %s\n", *tokenFile)
}

// getTokenWithFallback tries to get the code via HTTP, then falls back to manual entry
func getTokenWithFallback(config *oauth2.Config) *oauth2.Token {
	codeCh := make(chan string, 1)
	server := &http.Server{Addr: ":8080"}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err == nil {
			code := r.FormValue("code")
			if code != "" {
				fmt.Fprintf(w, "Auth complete! You can close this window.")
				codeCh <- code
				go func() { time.Sleep(1 * time.Second); server.Shutdown(context.Background()) }()
				return
			}
		}
		fmt.Fprintf(w, "No code found.")
	})
	go server.ListenAndServe()

	url := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Println("\n==== Google Drive OAuth2 Setup ====")
	fmt.Println("1. Open the following URL in your browser:")
	fmt.Println(url)
	fmt.Println("2. Authorize the app. If redirected to localhost, you can close the browser tab.")
	fmt.Println("3. If nothing happens in the terminal after 60 seconds, copy the 'code' parameter from the URL you were redirected to and paste it below.")
	openBrowser(url)

	var code string
	select {
	case code = <-codeCh:
		fmt.Println("\nReceived code via local server.")
	case <-time.After(60 * time.Second):
		fmt.Println("\nTimeout waiting for browser redirect.")
		fmt.Print("Paste the 'code' parameter from the redirected URL here: ")
		fmt.Scanln(&code)
	}

	tok, err := config.Exchange(context.Background(), code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

func saveToken(path string, token *oauth2.Token) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatalf("Unable to create token file: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func openBrowser(url string) {
	switch runtime.GOOS {
	case "linux":
		exec.Command("xdg-open", url).Start()
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		exec.Command("open", url).Start()
	}
}
