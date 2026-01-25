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

	info, err := client.GetAccountInfo()
	if err != nil {
		if apiErr, ok := err.(*tosend.Error); ok {
			fmt.Printf("API Error: %s (status: %d)\n", apiErr.Message, apiErr.StatusCode)
		} else {
			log.Fatal(err)
		}
		return
	}

	fmt.Println("=== Account Information ===")
	fmt.Println()
	fmt.Printf("Account:     %s\n", info.Account.Title)
	fmt.Printf("Plan:        %s\n", info.Account.PlanType)
	fmt.Printf("Status:      %s\n", info.Account.Status)
	fmt.Printf("Emails (24h): %d\n", info.Account.EmailsSentLast24Hrs)
	fmt.Printf("Emails (mo): %d\n", info.Account.EmailsUsageThisMonth)
	fmt.Println()

	fmt.Println("=== Domains ===")
	fmt.Println()
	if len(info.Domains) == 0 {
		fmt.Println("No domains configured")
	} else {
		for _, domain := range info.Domains {
			fmt.Printf("- %s (%s)\n", domain.DomainName, domain.VerificationStatus)
		}
	}
}
