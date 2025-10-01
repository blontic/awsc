# SWA Pagination Patterns

## Mandatory Pagination Rule
**ALL AWS list/describe operations MUST handle pagination**

## Common Pagination Patterns

### NextToken Pattern (EC2, SSO, Secrets Manager)
```go
var allItems []ItemType
var nextToken *string

for {
    result, err := client.ListItems(ctx, &service.ListItemsInput{
        NextToken: nextToken,
    })
    if err != nil {
        return nil, err
    }

    allItems = append(allItems, result.Items...)

    // Check if there are more pages
    if result.NextToken == nil {
        break
    }
    nextToken = result.NextToken
}
```

### Marker Pattern (RDS)
```go
var allItems []ItemType
var marker *string

for {
    result, err := client.DescribeItems(ctx, &service.DescribeItemsInput{
        Marker: marker,
    })
    if err != nil {
        return nil, err
    }

    allItems = append(allItems, result.Items...)

    // Check if there are more pages
    if result.Marker == nil {
        break
    }
    marker = result.Marker
}
```

## Services and Their Pagination

### SSO Services
- `ListAccounts` - Uses NextToken
- `ListAccountRoles` - Uses NextToken

### EC2 Services
- `DescribeInstances` - Uses NextToken
- `DescribeSecurityGroups` - Uses NextToken

### RDS Services
- `DescribeDBInstances` - Uses Marker

### Secrets Manager
- `ListSecrets` - Uses NextToken

### SSM Services
- `DescribeInstanceInformation` - Uses NextToken (if needed)

### CloudWatch Logs
- `DescribeLogGroups` - Uses NextToken
- `FilterLogEvents` - Uses NextToken

## Why Pagination is Critical
- **AWS limits page size** - Default is often 20-100 items per page
- **Large environments** - Organizations can have hundreds of accounts, instances, secrets
- **Data completeness** - Missing items leads to user confusion and bugs
- **Consistent behavior** - All operations should return complete results

## Implementation Requirements
- **Always use pagination loops** - Never assume single page response
- **Handle both auth errors and pagination** - Retry logic must work with pagination
- **Collect all pages before processing** - Don't process partial results
- **Use appropriate token field** - NextToken vs Marker depends on service