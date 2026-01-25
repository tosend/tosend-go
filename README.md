# ToSend Go SDK

Official Go SDK for the [ToSend](https://tosend.com) email API.

[![Go Reference](https://pkg.go.dev/badge/github.com/tosend/tosend-go.svg)](https://pkg.go.dev/github.com/tosend/tosend-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/tosend/tosend-go)](https://goreportcard.com/report/github.com/tosend/tosend-go)

## Installation

```bash
go get github.com/tosend/tosend-go
```

## Examples

See the [examples](examples) folder for complete working examples:

- [Send Email](examples/send_email/main.go) - Send a single email
- [Batch Send](examples/batch_send/main.go) - Send multiple emails in one request
- [Account Info](examples/account_info/main.go) - Get account information and domains

```bash
export TOSEND_API_KEY=tsend_your_api_key
go run examples/send_email/main.go
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/tosend/tosend-go"
)

func main() {
    client := tosend.New("tsend_your_api_key")

    resp, err := client.Send(&tosend.SendEmailRequest{
        From:    tosend.Address{Email: "hello@yourdomain.com", Name: "Your App"},
        To:      []tosend.Address{{Email: "user@example.com"}},
        Subject: "Welcome!",
        HTML:    "<h1>Hello!</h1><p>Thanks for signing up.</p>",
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Message ID:", resp.MessageID)
}
```

## Usage

### Send an Email

```go
resp, err := client.Send(&tosend.SendEmailRequest{
    From: tosend.Address{
        Email: "hello@yourdomain.com",
        Name:  "Your App",
    },
    To: []tosend.Address{
        {Email: "user@example.com"},
        {Email: "another@example.com", Name: "John Doe"},
    },
    Subject: "Hello!",
    HTML:    "<h1>Welcome</h1>",
    Text:    "Welcome", // optional
})
```

### With CC, BCC, and Reply-To

```go
resp, err := client.Send(&tosend.SendEmailRequest{
    From:    tosend.Address{Email: "hello@yourdomain.com"},
    To:      []tosend.Address{{Email: "user@example.com"}},
    CC:      []tosend.Address{{Email: "cc@example.com"}},
    BCC:     []tosend.Address{{Email: "bcc@example.com"}},
    ReplyTo: &tosend.Address{Email: "support@yourdomain.com", Name: "Support"},
    Subject: "Hello",
    HTML:    "<p>Hello World</p>",
})
```

### With Attachments

```go
import (
    "encoding/base64"
    "os"
)

content, _ := os.ReadFile("invoice.pdf")
encoded := base64.StdEncoding.EncodeToString(content)

resp, err := client.Send(&tosend.SendEmailRequest{
    From:    tosend.Address{Email: "hello@yourdomain.com"},
    To:      []tosend.Address{{Email: "user@example.com"}},
    Subject: "Your Invoice",
    HTML:    "<p>Please find your invoice attached.</p>",
    Attachments: []tosend.Attachment{
        {
            Type:    "application/pdf",
            Name:    "invoice.pdf",
            Content: encoded,
        },
    },
})
```

### Batch Sending

Send multiple emails in a single request:

```go
resp, err := client.Batch([]tosend.SendEmailRequest{
    {
        From:    tosend.Address{Email: "hello@yourdomain.com"},
        To:      []tosend.Address{{Email: "user1@example.com"}},
        Subject: "Hello User 1",
        HTML:    "<p>Welcome!</p>",
    },
    {
        From:    tosend.Address{Email: "hello@yourdomain.com"},
        To:      []tosend.Address{{Email: "user2@example.com"}},
        Subject: "Hello User 2",
        HTML:    "<p>Welcome!</p>",
    },
})

for _, result := range resp.Results {
    if result.Status == "success" {
        fmt.Println("Sent:", result.MessageID)
    } else {
        fmt.Println("Failed:", result.Message)
    }
}
```

### Get Account Info

```go
info, err := client.GetAccountInfo()
if err != nil {
    log.Fatal(err)
}

fmt.Println("Account:", info.Account.Title)
fmt.Println("Emails this month:", info.Account.EmailsUsageThisMonth)
fmt.Println("Emails last 24hrs:", info.Account.EmailsSentLast24Hrs)

for _, domain := range info.Domains {
    fmt.Printf("%s: %s\n", domain.DomainName, domain.VerificationStatus)
}
```

## Configuration

### With Options

```go
import "time"

client := tosend.New("tsend_your_api_key",
    tosend.WithBaseURL("https://api.tosend.com"),
    tosend.WithTimeout(60 * time.Second),
)
```

### Custom HTTP Client

```go
httpClient := &http.Client{
    Timeout: 60 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns: 10,
    },
}

client := tosend.New("tsend_your_api_key",
    tosend.WithHTTPClient(httpClient),
)
```

## Error Handling

```go
resp, err := client.Send(&tosend.SendEmailRequest{
    From:    tosend.Address{Email: "hello@yourdomain.com"},
    To:      []tosend.Address{{Email: "user@example.com"}},
    Subject: "Hello",
    HTML:    "<p>Hello</p>",
})

if err != nil {
    if apiErr, ok := err.(*tosend.Error); ok {
        fmt.Println("Error:", apiErr.Message)
        fmt.Println("Status:", apiErr.StatusCode)
        fmt.Println("Type:", apiErr.ErrorType)
        fmt.Println("Details:", apiErr.Errors)
    } else {
        fmt.Println("Error:", err)
    }
    return
}

fmt.Println("Sent:", resp.MessageID)
```

## Requirements

- Go 1.21 or higher

## License

MIT
