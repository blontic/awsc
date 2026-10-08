package aws

import (
	"context"
	"net/url"
	"strings"
	"testing"
)

func newTestConsoleManager(startURL, region string, opened *string) *ConsoleManager {
	m, _ := NewConsoleManager(context.Background(), ConsoleManagerOptions{
		StartURL:  startURL,
		AccountID: "471112661169",
		RoleName:  "cops-permset-teamadmin",
		Region:    region,
		Open:      func(u string) error { *opened = u; return nil },
	})
	return m
}

func TestConsoleRunOpen(t *testing.T) {
	tests := []struct {
		name, startURL, region, service, wantPortal, wantDestination string
	}{
		{"service page", "https://d-976710a35d.awsapps.com/start", "ap-southeast-2", "rds",
			"https://d-976710a35d.awsapps.com/start", "https://ap-southeast-2.console.aws.amazon.com/rds/home?region=ap-southeast-2"},
		{"home, dual-stack portal with trailing slash", "https://ssoins-82592fd9e404d1de.portal.ap-southeast-2.app.aws/", "us-east-1", "",
			"https://ssoins-82592fd9e404d1de.portal.ap-southeast-2.app.aws", "https://console.aws.amazon.com/console/home?region=us-east-1"},
		{"portal URL with fragment", "https://example.awsapps.com/start/#/", "eu-west-1", "ec2",
			"https://example.awsapps.com/start", "https://eu-west-1.console.aws.amazon.com/ec2/home?region=eu-west-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opened string
			if err := newTestConsoleManager(tt.startURL, tt.region, &opened).RunOpen(context.Background(), tt.service); err != nil {
				t.Fatal(err)
			}
			portal, fragment, ok := strings.Cut(opened, "/#/console?")
			if !ok || portal != tt.wantPortal {
				t.Fatalf("unexpected link %s", opened)
			}
			q, err := url.ParseQuery(fragment)
			if err != nil {
				t.Fatal(err)
			}
			if q.Get("account_id") != "471112661169" || q.Get("role_name") != "cops-permset-teamadmin" || q.Get("destination") != tt.wantDestination {
				t.Errorf("unexpected link parameters: %v", q)
			}
		})
	}
}

func TestConsoleRunOpen_Errors(t *testing.T) {
	tests := []struct{ name, startURL, region, service, want string }{
		{"bad service", "https://example.awsapps.com/start", "us-east-1", "ec2/../x?y", "invalid service"},
		{"bad region", "https://example.awsapps.com/start", "us-east-1.evil.com/", "ec2", "invalid region"},
		{"non-https start URL", "http://example.awsapps.com/start", "us-east-1", "ec2", "invalid SSO start URL"},
		{"missing start URL", "", "us-east-1", "ec2", "invalid SSO start URL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opened string
			err := newTestConsoleManager(tt.startURL, tt.region, &opened).RunOpen(context.Background(), tt.service)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("expected error containing %q, got %v", tt.want, err)
			}
			if opened != "" {
				t.Error("browser should not be opened")
			}
		})
	}
}

func TestConsoleHost(t *testing.T) {
	tests := []struct{ region, want string }{
		{"", "console.aws.amazon.com"},
		{"us-east-1", "console.aws.amazon.com"},
		{"eu-west-1", "eu-west-1.console.aws.amazon.com"},
		{"cn-north-1", "console.amazonaws.cn"},
		{"cn-northwest-1", "cn-northwest-1.console.amazonaws.cn"},
		{"us-gov-west-1", "console.amazonaws-us-gov.com"},
		{"us-gov-east-1", "us-gov-east-1.console.amazonaws-us-gov.com"},
	}
	for _, tt := range tests {
		if got := consoleHost(tt.region); got != tt.want {
			t.Errorf("consoleHost(%q) = %s, want %s", tt.region, got, tt.want)
		}
	}
}
