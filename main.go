package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/transip/gotransip/v6"
	"github.com/transip/gotransip/v6/domain"
)

func main() {
	accountName := os.Getenv("TRANSIP_ACCOUNT_NAME")
	privateKeyPath := os.Getenv("TRANSIP_PRIVATE_KEY_PATH")
	domainName := os.Getenv("DOMAIN")
	subdomainName := os.Getenv("SUBDOMAIN")
	if accountName == "" || privateKeyPath == "" {
		log.Fatal("TRANSIP_ACCOUNT_NAME and TRANSIP_PRIVATE_KEY_PATH environment variables must be set.")
	}
	ip, err := getPublicIPv4()
	if err != nil {
		log.Fatalf("Failed to get public IPv4 address: %v", err)
	}
	log.Printf("Current public IPv4 address: %s", ip)
	client, err := gotransip.NewClient(gotransip.ClientConfiguration{
		AccountName:    accountName,
		PrivateKeyPath: privateKeyPath,
	})
	if err != nil {
		log.Fatalf("Failed to create TransIP API client: %v", err)
	}
	domainRepo := domain.Repository{Client: client}
	dnsEntries, err := domainRepo.GetDNSEntries(domainName)
	if err != nil {
		log.Fatalf("Failed to get DNS entries for domain %s: %v", domainName, err)
	}
	foundRecord := false
	for _, dnsEntry := range dnsEntries {
		if dnsEntry.Name == subdomainName && dnsEntry.Type == "A" {
			log.Printf("Found existing A record for %s.%s: %s (TTL: %d)", subdomainName, domainName, dnsEntry.Content, dnsEntry.Expire)
			if dnsEntry.Content != ip {
				dnsEntry.Content = ip
				err = domainRepo.UpdateDNSEntry(domainName, dnsEntry)
				if err != nil {
					log.Fatalf("Failed to update A record for %s.%s to %s: %v", subdomainName, domainName, ip, err)
				}
				log.Printf("Successfully updated A record for %s.%s to %s", subdomainName, domainName, ip)
			} else {
				log.Printf("A record for %s.%s is already set to %s. No update needed.", subdomainName, domainName, ip)
			}
			foundRecord = true
			break
		}
	}
	if !foundRecord {
		newDNSEntry := domain.DNSEntry{
			Name:    subdomainName,
			Expire:  300,
			Type:    "A",
			Content: ip,
		}
		err = domainRepo.AddDNSEntry(domainName, newDNSEntry)
		if err != nil {
			log.Fatalf("Failed to add new A record for %s.%s to %s: %v", subdomainName, domainName, ip, err)
		}
		log.Printf("Successfully added new A record for %s.%s to %s", subdomainName, domainName, ip)
	}
}

func getPublicIPv4() (string, error) {
	resp, err := http.Get("https://api.ipify.org")
	if err != nil {
		return "", fmt.Errorf("failed to make HTTP request to ipify.org: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ipify.org returned non-OK status: %d %s", resp.StatusCode, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body from ipify.org: %w", err)
	}
	return string(body), nil
}
