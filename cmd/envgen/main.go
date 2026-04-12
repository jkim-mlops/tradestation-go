package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

func main() {
	idParam := flag.String("id", "/tradestation/client-id", "SSM parameter name for client ID")
	secretParam := flag.String("secret", "/tradestation/client-secret", "SSM parameter name for client secret")
	refreshParam := flag.String("refresh", "/tradestation/refresh-token", "SSM parameter name for refresh token")
	out := flag.String("out", ".env", "output file path")
	flag.Parse()

	profile := os.Getenv("AWS_PROFILE")
	if profile == "" {
		profile = "joe-prod"
	}

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(profile))
	if err != nil {
		log.Fatalf("load aws config (profile=%s): %v", profile, err)
	}

	resp, err := ssm.NewFromConfig(cfg).GetParameters(ctx, &ssm.GetParametersInput{
		Names:          []string{*idParam, *secretParam, *refreshParam},
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		log.Fatalf("ssm get parameters: %v", err)
	}
	if len(resp.InvalidParameters) > 0 {
		log.Fatalf("ssm parameters not found: %v", resp.InvalidParameters)
	}

	values := make(map[string]string, len(resp.Parameters))
	for _, p := range resp.Parameters {
		values[aws.ToString(p.Name)] = aws.ToString(p.Value)
	}

	body := fmt.Sprintf(
		"TRADESTATION_CLIENT_ID=%s\nTRADESTATION_CLIENT_SECRET=%s\nTRADESTATION_REFRESH_TOKEN=%s\n",
		values[*idParam], values[*secretParam], values[*refreshParam])

	if err := os.WriteFile(*out, []byte(body), 0o600); err != nil {
		log.Fatalf("write %s: %v", *out, err)
	}
	fmt.Printf("wrote %s\n", *out)
}
