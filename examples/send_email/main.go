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

	resp, err := client.Send(&tosend.SendEmailRequest{
		From: tosend.Address{
			Email: "hello@yourdomain.com",
			Name:  "Your App",
		},
		To: []tosend.Address{
			{Email: "user@example.com", Name: "John Doe"},
		},
		Subject: "Welcome to ToSend!",
		HTML:    "<h1>Hello!</h1><p>Thanks for signing up.</p>",
		Text:    "Hello! Thanks for signing up.",
	})

	if err != nil {
		if apiErr, ok := err.(*tosend.Error); ok {
			fmt.Printf("API Error: %s (status: %d)\n", apiErr.Message, apiErr.StatusCode)
		} else {
			log.Fatal(err)
		}
		return
	}

	fmt.Println("Email sent successfully!")
	fmt.Println("Message ID:", resp.MessageID)
}
