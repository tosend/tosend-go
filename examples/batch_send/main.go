package main

import (
	"fmt"
	"log"
	"os"

	"github.com/tosend/tosend-go"
)

func main() {
	apiKey := os.Getenv("TOSEND_API_KEY")
	if apiKey == "" {
		log.Fatal("TOSEND_API_KEY environment variable is required")
	}

	client := tosend.New(apiKey)

	emails := []tosend.SendEmailRequest{
		{
			From:    tosend.Address{Email: "hello@yourdomain.com", Name: "Your App"},
			To:      []tosend.Address{{Email: "user1@example.com"}},
			Subject: "Hello User 1",
			HTML:    "<p>Welcome to our platform!</p>",
		},
		{
			From:    tosend.Address{Email: "hello@yourdomain.com", Name: "Your App"},
			To:      []tosend.Address{{Email: "user2@example.com"}},
			Subject: "Hello User 2",
			HTML:    "<p>Welcome to our platform!</p>",
		},
		{
			From:    tosend.Address{Email: "hello@yourdomain.com", Name: "Your App"},
			To:      []tosend.Address{{Email: "user3@example.com"}},
			Subject: "Hello User 3",
			HTML:    "<p>Welcome to our platform!</p>",
		},
	}

	resp, err := client.Batch(emails)
	if err != nil {
		if apiErr, ok := err.(*tosend.Error); ok {
			fmt.Printf("API Error: %s (status: %d)\n", apiErr.Message, apiErr.StatusCode)
		} else {
			log.Fatal(err)
		}
		return
	}

	fmt.Printf("Batch sent! Processing %d emails:\n\n", len(resp.Results))

	for i, result := range resp.Results {
		if result.Status == "success" {
			fmt.Printf("[%d] Success - Message ID: %s\n", i+1, result.MessageID)
		} else {
			fmt.Printf("[%d] Failed - %s\n", i+1, result.Message)
		}
	}
}
