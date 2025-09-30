# SWA Constructor Patterns

## Mandatory Constructor Pattern
**ALWAYS use optional parameter pattern for manager constructors**

### Pattern Structure
```go
type ManagerOptions struct {
    Client    ClientInterface
    Region    string
    // ... other injectable dependencies
}

func NewManager(ctx context.Context, opts ...ManagerOptions) (*Manager, error) {
    if len(opts) > 0 && opts[0].Client != nil {
        // Use provided clients (for testing)
        return &Manager{
            client: opts[0].Client,
            region: opts[0].Region,
        }, nil
    }
    
    // Production path
    cfg, err := swaconfig.LoadSWAConfigWithProfile(ctx)
    if err != nil {
        return nil, err
    }

    return &Manager{
        client: service.NewFromConfig(cfg),
        region: cfg.Region,
    }, nil
}
```

### Usage Examples

**Production usage:**
```go
manager, err := NewRDSManager(ctx)
```

**Test usage:**
```go
manager, err := NewRDSManager(ctx, RDSManagerOptions{
    RDSClient: mockRDS,
    EC2Client: mockEC2,
    Region:    "us-east-1",
})
```

## Rules

### NEVER Do This
- ❌ `NewManagerWithClients()` functions
- ❌ `NewManagerForTesting()` functions  
- ❌ Separate test-only constructors
- ❌ Build tags for test constructors

### ALWAYS Do This
- ✅ Single constructor with optional parameters
- ✅ `ManagerOptions` struct for dependencies
- ✅ Variadic parameters: `opts ...ManagerOptions`
- ✅ Check `len(opts) > 0 && opts[0].Client != nil` for test path
- ✅ Load real clients in production path

## Benefits
- Clean separation of test and production code
- No test-only code in production files
- Go-idiomatic optional parameter pattern
- Consistent across all managers
- Easy to extend with new dependencies

## Enforcement
- **Code reviews must reject** separate test constructors
- **All new managers must follow** this pattern
- **Existing managers should be updated** to this pattern
- **Tests must use** the optional parameter approach