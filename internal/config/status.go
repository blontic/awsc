package config

import (
	"fmt"
	"os"
	"strings"
)

// Status is the org, account and role that commands in this terminal use.
type Status struct {
	Org         string
	AccountName string
	AccountID   string
	RoleName    string
	Profile     string
	Region      string
}

// CurrentStatus works out, without changing any files, the org, account and
// role that commands in this terminal would use. flagOrg and region are the
// --org and --region flags ("" if not given).
func CurrentStatus(flagOrg, region string) (*Status, error) {
	cfg, err := ReadFileConfig()
	if err != nil {
		return nil, err
	}
	if len(cfg.Orgs) == 0 {
		return nil, fmt.Errorf("no orgs configured; add one with 'awsc config add'")
	}
	org, err := resolveActiveOrg(cfg, flagOrg)
	if err != nil {
		return nil, err
	}
	if org == "" {
		return nil, fmt.Errorf("multiple orgs configured and no default set; use --org <name> or 'awsc config use <name>'")
	}

	status := &Status{Org: org, Region: region}
	if status.Region == "" {
		status.Region = cfg.Orgs[org].DefaultRegion
	}

	if profile := os.Getenv("AWSC_PROFILE"); profile != "" {
		accountID, roleName, accountName, found, err := lookupProfileDetails(profile)
		if err != nil {
			return nil, err
		}
		if !found || accountID == "" || roleName == "" {
			return nil, fmt.Errorf("profile %q (AWSC_PROFILE) is not an SSO profile in ~/.aws/config", profile)
		}
		status.Profile, status.AccountID, status.RoleName, status.AccountName = profile, accountID, roleName, accountName
		return status, nil
	}

	session, err := GetCurrentSession()
	if err != nil || session.Org != org {
		return nil, fmt.Errorf("no account selected in this terminal for org %q; run 'awsc login'", org)
	}
	status.Profile = session.ProfileName
	status.AccountID = session.AccountID
	status.AccountName = session.AccountName
	status.RoleName = session.RoleName
	return status, nil
}

// lookupProfileDetails is lookupProfile plus the account name from the
// "# Account:" comment awsc writes into its profiles ("" for other profiles).
func lookupProfileDetails(name string) (accountID, roleName, accountName string, found bool, err error) {
	accountID, roleName, found, err = lookupProfile(name)
	if err != nil || !found {
		return accountID, roleName, "", found, err
	}
	path, err := awsConfigPath()
	if err != nil {
		return "", "", "", false, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", "", false, err
	}
	for _, s := range parseSections(string(data)) {
		if s.name != "profile "+name {
			continue
		}
		for _, line := range s.lines[1:] {
			if v, ok := strings.CutPrefix(strings.TrimSpace(line), "# Account:"); ok {
				accountName = strings.TrimSpace(v)
				break
			}
		}
		break
	}
	return accountID, roleName, accountName, true, nil
}
