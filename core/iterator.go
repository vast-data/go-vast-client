package core

import (
	"context"
	"fmt"
	"strings"
)

// ######################################################
//              ITERATOR INTERFACES
// ######################################################

// Iterator provides an interface for iterating over paginated or non-paginated API results.
// It abstracts away the differences between paginated resources (with next/previous links)
// and non-paginated resources (flat lists).
type Iterator interface {
	// Next advances to the next page and returns the records and any error.
	// Returns empty RecordSet when there are no more pages.
	Next() (RecordSet, error)

	// Previous moves to the previous page and returns the records and any error.
	// Returns empty RecordSet when there is no previous page.
	Previous() (RecordSet, error)

	// HasNext returns true if there is a next page available.
	HasNext() bool

	// HasPrevious returns true if there is a previous page available.
	HasPrevious() bool

	// Count returns the total count of items (if available from pagination metadata).
	// Returns -1 if count information is not available.
	Count() int

	// PageSize returns the current page size.
	PageSize() int

	// Reset resets the iterator to the first page and returns the first page records.
	Reset() (RecordSet, error)

	// All fetches all remaining pages and returns all records as a single RecordSet.
	// This should be used with caution for large datasets.
	All() (RecordSet, error)
}

// ######################################################
//              RESOURCE ITERATOR IMPLEMENTATION
// ######################################################

// ResourceIterator implements the Iterator interface for VAST API resources.
// It makes raw HTTP requests to preserve pagination metadata from the API.
type ResourceIterator struct {
	resource     VastResourceAPIWithContext
	ctx          context.Context
	initialQuery Params
	pageSize     int

	current     RecordSet
	nextURL     *string
	previousURL *string
	totalCount  int
	currentPage int
	err         error
	initialized bool
}

// NewResourceIterator creates an iterator that makes raw HTTP requests to preserve pagination metadata.
// If pageSize is 0 or negative, uses the session's configured PageSize (default: 0 means no page_size param sent).
func NewResourceIterator(ctx context.Context, resource VastResourceAPIWithContext, params Params, pageSize int) Iterator {
	if pageSize <= 0 {
		// Get default page size from session config
		config := resource.Session().GetConfig()
		pageSize = config.PageSize // May be 0, which means don't send page_size param
	}

	if params == nil {
		params = make(Params)
	}
	// Only add page_size to params if pageSize > 0
	if _, exists := params["page_size"]; !exists && pageSize > 0 {
		params["page_size"] = pageSize
	}

	return &ResourceIterator{
		resource:     resource,
		ctx:          ctx,
		initialQuery: params,
		pageSize:     pageSize,
		totalCount:   -1,
		currentPage:  0,
		initialized:  false,
	}
}

// fetchPage makes a raw HTTP request and processes the pagination envelope.
func (it *ResourceIterator) fetchPage(url string, params Params) error {
	session := it.resource.Session()

	// Make raw HTTP request
	var response Renderable
	var err error

	if url != "" {
		// Use the full URL for next/previous navigation
		response, err = session.Get(it.ctx, url, nil, nil)
	} else {
		// Use resource path with params for first request
		resourcePath := it.resource.GetResourcePath()
		query := params.ToQuery()
		fullURL, buildErr := buildUrl(session, resourcePath, query, session.GetConfig().ApiVersion)
		if buildErr != nil {
			return buildErr
		}
		response, err = session.Get(it.ctx, fullURL, nil, nil)
	}

	if err != nil {
		return err
	}

	// Check if response is a pagination envelope
	if record, ok := response.(Record); ok {
		return it.processPaginationEnvelope(record)
	}

	// If it's already a RecordSet, it's been unwrapped - treat as non-paginated
	if recordSet, ok := response.(RecordSet); ok {
		it.current = recordSet
		it.nextURL = nil
		it.previousURL = nil
		it.totalCount = len(recordSet)

		// Set resource type on all records for consistency
		resourceType := it.resource.GetResourceType()
		if resourceType != "Dummy" {
			if err := setResourceKey(it.current, resourceType); err != nil {
				return err
			}
		}
		return nil
	}

	return fmt.Errorf("unexpected response type: %T", response)
}

// processPaginationEnvelope extracts data from a pagination envelope using ToRecordSet.
func (it *ResourceIterator) processPaginationEnvelope(envelope Record) error {
	// Check if this is a pagination envelope
	_, hasResults := envelope["results"]
	_, hasCount := envelope["count"]
	_, hasNext := envelope["next"]
	_, hasPrev := envelope["previous"]

	if hasResults && hasCount && hasNext && hasPrev {
		// This is a paginated response - use ToRecordSet to convert results
		if resultsRaw, ok := envelope["results"]; ok {
			// Try []map[string]any first
			if resultsMapList, ok := resultsRaw.([]map[string]any); ok {
				recordSet, err := ToRecordSet(resultsMapList)
				if err != nil {
					return fmt.Errorf("failed to convert results to RecordSet: %w", err)
				}
				it.current = recordSet
				// Set resource type on all records for consistency
				resourceType := it.resource.GetResourceType()
				if resourceType != "Dummy" {
					if err := setResourceKey(it.current, resourceType); err != nil {
						return err
					}
				}
			} else if resultsList, ok := resultsRaw.([]any); ok {
				// Convert []any to []map[string]any
				converted := make([]map[string]any, 0, len(resultsList))
				for _, item := range resultsList {
					if record, ok := item.(map[string]any); ok {
						converted = append(converted, record)
					} else {
						return fmt.Errorf("unexpected type in results array: %T", item)
					}
				}
				recordSet, err := ToRecordSet(converted)
				if err != nil {
					return fmt.Errorf("failed to convert results to RecordSet: %w", err)
				}
				it.current = recordSet
				// Set resource type on all records for consistency
				resourceType := it.resource.GetResourceType()
				if resourceType != "Dummy" {
					if err := setResourceKey(it.current, resourceType); err != nil {
						return err
					}
				}
			} else {
				return fmt.Errorf("unexpected type for results field: %T", resultsRaw)
			}
		}

		if count, ok := envelope["count"]; ok {
			if countFloat, ok := count.(float64); ok {
				it.totalCount = int(countFloat)
			} else if countInt, ok := count.(int); ok {
				it.totalCount = countInt
			}
		}

		if next, ok := envelope["next"]; ok {
			if next == nil {
				it.nextURL = nil
			} else if nextStr, ok := next.(string); ok && nextStr != "" {
				it.nextURL = &nextStr
			} else {
				it.nextURL = nil
			}
		}

		if prev, ok := envelope["previous"]; ok {
			if prev == nil {
				it.previousURL = nil
			} else if prevStr, ok := prev.(string); ok && prevStr != "" {
				it.previousURL = &prevStr
			} else {
				it.previousURL = nil
			}
		}

		return nil
	}

	// Not a pagination envelope - treat the record itself as the result
	it.current = RecordSet{envelope}
	it.nextURL = nil
	it.previousURL = nil
	it.totalCount = 1

	// Set resource type on all records for consistency
	resourceType := it.resource.GetResourceType()
	if resourceType != "Dummy" {
		if err := setResourceKey(it.current, resourceType); err != nil {
			return err
		}
	}
	return nil
}

// Next advances to the next page and returns the records and any error.
func (it *ResourceIterator) Next() (RecordSet, error) {
	if !it.initialized {
		it.err = it.fetchPage("", it.initialQuery)
		it.initialized = true
		if it.err != nil {
			return RecordSet{}, it.err
		}
		return it.current, nil
	}

	if !it.HasNext() {
		return RecordSet{}, nil
	}

	it.err = it.fetchPage(*it.nextURL, nil)
	if it.err != nil {
		return RecordSet{}, it.err
	}

	it.currentPage++
	return it.current, nil
}

// Previous moves to the previous page and returns the records and any error.
func (it *ResourceIterator) Previous() (RecordSet, error) {
	if !it.initialized {
		it.err = fmt.Errorf("iterator not initialized, call Next() first")
		return RecordSet{}, it.err
	}

	if !it.HasPrevious() {
		return RecordSet{}, nil
	}

	it.err = it.fetchPage(*it.previousURL, nil)
	if it.err != nil {
		return RecordSet{}, it.err
	}

	it.currentPage--
	return it.current, nil
}

// HasNext returns true if there is a next page.
func (it *ResourceIterator) HasNext() bool {
	if !it.initialized {
		return true
	}
	return it.nextURL != nil && *it.nextURL != ""
}

// HasPrevious returns true if there is a previous page.
func (it *ResourceIterator) HasPrevious() bool {
	if !it.initialized {
		return false
	}
	return it.previousURL != nil && *it.previousURL != ""
}

// Count returns the total count of items.
func (it *ResourceIterator) Count() int {
	return it.totalCount
}

// PageSize returns the page size.
func (it *ResourceIterator) PageSize() int {
	return it.pageSize
}

// String returns a formatted string representation of the iterator state.
func (it *ResourceIterator) String() string {
	var sb strings.Builder

	sb.WriteString("ResourceIterator {\n")
	sb.WriteString(fmt.Sprintf("  Initialized:   %v\n", it.initialized))
	sb.WriteString(fmt.Sprintf("  Current Page:  %d\n", it.currentPage))
	sb.WriteString(fmt.Sprintf("  Page Size:     %d\n", it.pageSize))
	sb.WriteString(fmt.Sprintf("  Total Count:   %d\n", it.totalCount))

	// Show current records with count
	if it.current != nil && len(it.current) > 0 {
		sb.WriteString(fmt.Sprintf("  Current:       [... (%d items)]\n", len(it.current)))
	} else {
		sb.WriteString("  Current:       []\n")
	}

	// Show next URL
	if it.nextURL != nil && *it.nextURL != "" {
		sb.WriteString(fmt.Sprintf("  Next URL:      %s\n", *it.nextURL))
	} else {
		sb.WriteString("  Next URL:      <none>\n")
	}

	// Show previous URL
	if it.previousURL != nil && *it.previousURL != "" {
		sb.WriteString(fmt.Sprintf("  Previous URL:  %s\n", *it.previousURL))
	} else {
		sb.WriteString("  Previous URL:  <none>\n")
	}

	// Show error if any
	if it.err != nil {
		sb.WriteString(fmt.Sprintf("  Error:         %v\n", it.err))
	}

	sb.WriteString("}")
	return sb.String()
}

// Reset resets the iterator to the first page and returns the first page records.
func (it *ResourceIterator) Reset() (RecordSet, error) {
	it.initialized = false
	it.current = nil
	it.nextURL = nil
	it.previousURL = nil
	it.currentPage = 0
	it.err = nil
	it.totalCount = -1

	return it.Next()
}

// All fetches all pages and returns all records.
func (it *ResourceIterator) All() (RecordSet, error) {
	var allRecords RecordSet

	if !it.initialized {
		records, err := it.Next()
		if err != nil {
			return nil, err
		}
		allRecords = append(allRecords, records...)
	} else {
		// Include current page if already initialized
		allRecords = append(allRecords, it.current...)
	}

	for it.HasNext() {
		records, err := it.Next()
		if err != nil {
			return nil, err
		}
		allRecords = append(allRecords, records...)
	}

	return allRecords, nil
}
