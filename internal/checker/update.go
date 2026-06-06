package checker

import (
	"context"
	"errors"

	"github.com/cloudflare/cloudflare-go"
)

func updateDNSIPv4(api *cloudflare.API, zoneID string, recordName string, currentIP string) error {
	ctx := context.Background()

	records, _, err := api.ListDNSRecords(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.ListDNSRecordsParams{
		Type: "A",
		Name: recordName,
	})
	if err != nil {
		return errors.New("failed to list A DNS records")
	}

	if len(records) == 0 {
		_, err = api.CreateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.CreateDNSRecordParams{
			Type:    "A",
			Name:    recordName,
			Content: currentIP,
			TTL:     60,
			Proxied: new(false),
		})
	} else {
		_, err = api.UpdateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.UpdateDNSRecordParams{
			ID:      records[0].ID,
			Type:    "A",
			Name:    recordName,
			Content: currentIP,
		})
	}

	if err != nil {
		return errors.New("failed to update A DNS records")
	}
	return nil
}

func updateDNSIPv6(api *cloudflare.API, zoneID string, recordName string, currentIP string) error {
	ctx := context.Background()

	records, _, err := api.ListDNSRecords(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.ListDNSRecordsParams{
		Type: "AAAA",
		Name: recordName,
	})
	if err != nil {
		return errors.New("failed to list AAAA DNS records")
	}

	if len(records) == 0 {
		_, err = api.CreateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.CreateDNSRecordParams{
			Type:    "AAAA",
			Name:    recordName,
			Content: currentIP,
			TTL:     60,
			Proxied: new(false),
		})
	} else {
		_, err = api.UpdateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.UpdateDNSRecordParams{
			ID:      records[0].ID,
			Type:    "AAAA",
			Name:    recordName,
			Content: currentIP,
		})
	}

	if err != nil {
		return errors.New("failed to update AAAA DNS records")
	}
	return nil
}
