# Iterators

## Overview

Iterators provide low-level access to paginated API responses. **For most use cases, you should use `List()` or `ListWithContext()` methods instead**, which automatically fetch all pages and return complete results.

Use iterators only when you need fine-grained control over pagination, such as:

- Processing very large datasets page-by-page to limit memory usage
- Implementing custom pagination logic
- Early termination based on page contents

## Basic Usage

### Simple Iteration

```go
// Create an iterator with page size of 50
iter := resource.GetIterator(core.Params{"name__contains": "test"}, 50)

// Iterate through pages
for {
    records, err := iter.Next()
    if err != nil {
        log.Fatalf("Error: %v", err)
    }
    if len(records) == 0 {
        break // No more pages
    }
    
    // Process current page
    for _, record := range records {
        fmt.Printf("ID: %v, Name: %v\n", record["id"], record["name"])
    }
}
```

### Using Context

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

iter := resource.GetIteratorWithContext(ctx, core.Params{}, 100)

for {
    records, err := iter.Next()
    if err != nil {
        log.Fatalf("Error: %v", err)
    }
    if len(records) == 0 {
        break
    }
    // Process records...
}
```

### Fetch All Pages at Once

```go
iter := resource.GetIterator(core.Params{}, 100)

// Fetch all pages into a single RecordSet
allRecords, err := iter.All()
if err != nil {
    log.Fatalf("Error: %v", err)
}

fmt.Printf("Total records: %d\n", len(allRecords))
```

## Iterator Methods

```go
type Iterator interface {
    Next() (RecordSet, error)          // Advance to next page
    Previous() (RecordSet, error)      // Go to previous page
    HasNext() bool                     // Check if next page exists
    HasPrevious() bool                 // Check if previous page exists
    Count() int                        // Total count (-1 if unknown)
    PageSize() int                     // Current page size
    Reset() (RecordSet, error)         // Reset to first page
    All() (RecordSet, error)           // Fetch all remaining pages
}
```

## Configuration

### Global Page Size

```go
config := &core.VMSConfig{
    Host:     "vast.example.com",
    ApiToken: "your-token",
    PageSize: 100,  // Default for all iterators (0 = server default)
}
```

### Per-Iterator Page Size

```go
// Use config default (0 = no page_size param, server decides)
iter := resource.GetIterator(params, 0)

// Override with specific page size
iter := resource.GetIterator(params, 50)
```

## Prefer List() Over Iterators

**Recommended approach** for most use cases:

```go
// Simple and straightforward - fetches all pages automatically
records, err := resource.List(core.Params{"status": "active"})
if err != nil {
    log.Fatalf("Error: %v", err)
}

// Process all records
for _, record := range records {
    fmt.Printf("Processing: %v\n", record["name"])
}
```

**Use iterators only when needed**:

```go
// Example: Process large dataset page-by-page to limit memory
iter := resource.GetIterator(core.Params{}, 1000)
processedCount := 0

for {
    records, err := iter.Next()
    if err != nil || len(records) == 0 {
        break
    }
    
    for _, record := range records {
        // Process one record at a time
        processRecord(record)
        processedCount++
    }
    
    log.Printf("Processed %d records so far...\n", processedCount)
}
```

## API Support

Iterators handle both paginated and non-paginated endpoints transparently:
- **Paginated responses**: Responses with `results`, `count`, `next`, `previous` fields
- **Non-paginated responses**: Flat arrays are treated as a single page

The iterator automatically detects the response format and adapts accordingly.
