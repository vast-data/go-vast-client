package core

import (
	"context"
	"fmt"
	"strings"
)

// newIteratorFn is the rest-owned constructor signature for pagination iterators.
type newIteratorFn func(ctx context.Context, resource VastResourceAPIWithContext, params Params, pageSize int) Iterator

// recordSetFromListField converts a JSON list field ([]any or []map[string]any) to RecordSet.
func recordSetFromListField(raw any, fieldName string) (RecordSet, error) {
	if resultsMapList, ok := raw.([]map[string]any); ok {
		return ToRecordSet(resultsMapList)
	}
	if resultsList, ok := raw.([]any); ok {
		converted := make([]map[string]any, 0, len(resultsList))
		for _, item := range resultsList {
			if record, ok := item.(map[string]any); ok {
				converted = append(converted, record)
			} else {
				return nil, fmt.Errorf("unexpected type in %s array: %T", fieldName, item)
			}
		}
		return ToRecordSet(converted)
	}
	return nil, fmt.Errorf("unexpected type for %s field: %T", fieldName, raw)
}

func optionalURLString(v any) *string {
	if v == nil {
		return nil
	}
	if s, ok := v.(string); ok && s != "" {
		return &s
	}
	return nil
}

// resolvePageSize returns pageSize, falling back to session config when <= 0.
func resolvePageSize(resource VastResourceAPIWithContext, pageSize int) int {
	if pageSize <= 0 {
		return resource.Session().GetConfig().PageSize
	}
	return pageSize
}

// ######################################################
//              VMS ITERATOR
// ######################################################

// VmsIterator implements Iterator for VMS (DRF-style) list responses:
//
//	{ "results": [...], "count": N, "next": "<url>|null", "previous": "<url>|null" }
//
// Non-paginated responses (flat RecordSet or a single Record) are also supported.
// Selected internally for resources whose rest apiRoot is empty (main VMS).
type VmsIterator struct {
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

// NewVmsIterator creates a VMS DRF pagination iterator.
func NewVmsIterator(ctx context.Context, resource VastResourceAPIWithContext, params Params, pageSize int) Iterator {
	pageSize = resolvePageSize(resource, pageSize)

	if params == nil {
		params = make(Params)
	}
	if _, exists := params["page_size"]; !exists && pageSize > 0 {
		params["page_size"] = pageSize
	}

	return &VmsIterator{
		resource:     resource,
		ctx:          ctx,
		initialQuery: params,
		pageSize:     pageSize,
		totalCount:   -1,
	}
}

func (it *VmsIterator) fetchPage(url string, params Params) error {
	session := it.resource.Session()

	var response Renderable
	var err error

	if url != "" {
		response, err = session.Get(it.ctx, url, nil, nil)
	} else {
		resourcePath := it.resource.GetResourcePath()
		fullURL, buildErr := buildUrl(session, resourcePath, params.ToQuery(), session.GetConfig().ApiVersion, it.resource.GetApiRoot())
		if buildErr != nil {
			return buildErr
		}
		response, err = session.Get(it.ctx, fullURL, nil, nil)
	}
	if err != nil {
		return err
	}

	if record, ok := response.(Record); ok {
		return it.processEnvelope(record)
	}
	if recordSet, ok := response.(RecordSet); ok {
		it.current = recordSet
		it.nextURL = nil
		it.previousURL = nil
		it.totalCount = len(recordSet)
		return nil
	}
	return fmt.Errorf("unexpected response type: %T", response)
}

func (it *VmsIterator) processEnvelope(envelope Record) error {
	_, hasResults := envelope["results"]
	_, hasCount := envelope["count"]
	if hasResults && hasCount {
		recordSet, err := recordSetFromListField(envelope["results"], "results")
		if err != nil {
			return err
		}
		it.current = recordSet
		if count, ok := envelope["count"]; ok {
			switch c := count.(type) {
			case float64:
				it.totalCount = int(c)
			case int:
				it.totalCount = c
			}
		}
		it.nextURL = optionalURLString(envelope["next"])
		it.previousURL = optionalURLString(envelope["previous"])
		return nil
	}

	// Single object / non-list envelope
	it.current = RecordSet{envelope}
	it.nextURL = nil
	it.previousURL = nil
	it.totalCount = 1
	return nil
}

func (it *VmsIterator) Next() (RecordSet, error) {
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

func (it *VmsIterator) Previous() (RecordSet, error) {
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

func (it *VmsIterator) HasNext() bool {
	if !it.initialized {
		return true
	}
	return it.nextURL != nil && *it.nextURL != ""
}

func (it *VmsIterator) HasPrevious() bool {
	if !it.initialized {
		return false
	}
	return it.previousURL != nil && *it.previousURL != ""
}

func (it *VmsIterator) Count() int    { return it.totalCount }
func (it *VmsIterator) PageSize() int { return it.pageSize }

func (it *VmsIterator) Reset() (RecordSet, error) {
	it.initialized = false
	it.current = nil
	it.nextURL = nil
	it.previousURL = nil
	it.currentPage = 0
	it.err = nil
	it.totalCount = -1
	return it.Next()
}

func (it *VmsIterator) All() (RecordSet, error) {
	var allRecords RecordSet
	if !it.initialized {
		records, err := it.Next()
		if err != nil {
			return nil, err
		}
		allRecords = append(allRecords, records...)
	} else {
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

func (it *VmsIterator) String() string {
	var sb strings.Builder
	sb.WriteString("VmsIterator {\n")
	sb.WriteString(fmt.Sprintf("  Initialized:   %v\n", it.initialized))
	sb.WriteString(fmt.Sprintf("  Current Page:  %d\n", it.currentPage))
	sb.WriteString(fmt.Sprintf("  Page Size:     %d\n", it.pageSize))
	sb.WriteString(fmt.Sprintf("  Total Count:   %d\n", it.totalCount))
	if len(it.current) > 0 {
		sb.WriteString(fmt.Sprintf("  Current:       [... (%d items)]\n", len(it.current)))
	} else {
		sb.WriteString("  Current:       []\n")
	}
	if it.nextURL != nil && *it.nextURL != "" {
		sb.WriteString(fmt.Sprintf("  Next URL:      %s\n", *it.nextURL))
	} else {
		sb.WriteString("  Next URL:      <none>\n")
	}
	if it.previousURL != nil && *it.previousURL != "" {
		sb.WriteString(fmt.Sprintf("  Previous URL:  %s\n", *it.previousURL))
	} else {
		sb.WriteString("  Previous URL:  <none>\n")
	}
	if it.err != nil {
		sb.WriteString(fmt.Sprintf("  Error:         %v\n", it.err))
	}
	sb.WriteString("}")
	return sb.String()
}

// ######################################################
//              DATA ENGINE ITERATOR
// ######################################################

// DataEngineIterator implements Iterator for DataEngine (serverless) list responses:
//
//	{ "data": [...], "pagination": { "next_cursor": "...", "previous_cursor": "..." } }
//
// Next/previous navigation uses synthesized URLs with ?cursor=... (cursors are not absolute URLs).
// An empty data page ends iteration even if cursors are still present.
// Selected internally for resources whose rest apiRoot is "serverless".
type DataEngineIterator struct {
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

// NewDataEngineIterator creates a DataEngine cursor pagination iterator.
// When pageSize > 0, query param "limit" is set unless already present
// (same in-place mutation behavior as the original ResourceIterator for page_size).
func NewDataEngineIterator(ctx context.Context, resource VastResourceAPIWithContext, params Params, pageSize int) Iterator {
	pageSize = resolvePageSize(resource, pageSize)

	if params == nil {
		params = make(Params)
	}
	if _, exists := params["limit"]; !exists && pageSize > 0 {
		params["limit"] = pageSize
	}

	return &DataEngineIterator{
		resource:     resource,
		ctx:          ctx,
		initialQuery: params,
		pageSize:     pageSize,
		totalCount:   -1,
	}
}

func (it *DataEngineIterator) fetchPage(url string, params Params) error {
	session := it.resource.Session()

	var response Renderable
	var err error

	if url != "" {
		response, err = session.Get(it.ctx, url, nil, nil)
	} else {
		resourcePath := it.resource.GetResourcePath()
		fullURL, buildErr := buildUrl(session, resourcePath, params.ToQuery(), session.GetConfig().ApiVersion, it.resource.GetApiRoot())
		if buildErr != nil {
			return buildErr
		}
		response, err = session.Get(it.ctx, fullURL, nil, nil)
	}
	if err != nil {
		return err
	}

	if record, ok := response.(Record); ok {
		return it.processEnvelope(record)
	}
	if recordSet, ok := response.(RecordSet); ok {
		// Unexpected for DE, but tolerate a flat list.
		it.current = recordSet
		it.nextURL = nil
		it.previousURL = nil
		it.totalCount = -1
		return nil
	}
	return fmt.Errorf("unexpected response type: %T", response)
}

func (it *DataEngineIterator) processEnvelope(envelope Record) error {
	if _, hasData := envelope["data"]; !hasData {
		// Not a DE list envelope — treat as a single record.
		it.current = RecordSet{envelope}
		it.nextURL = nil
		it.previousURL = nil
		it.totalCount = -1
		return nil
	}

	recordSet, err := recordSetFromListField(envelope["data"], "data")
	if err != nil {
		return err
	}
	it.current = recordSet
	it.totalCount = -1

	// Empty page means end of cursor walk (next_cursor may still be present).
	if len(it.current) == 0 {
		it.nextURL = nil
		it.previousURL = nil
		return nil
	}

	nextCursor, prevCursor := extractDataEngineCursors(envelope["pagination"])
	it.nextURL, err = it.buildCursorURL(nextCursor)
	if err != nil {
		return err
	}
	it.previousURL, err = it.buildCursorURL(prevCursor)
	return err
}

func extractDataEngineCursors(paginationRaw any) (next, prev string) {
	pagination, ok := paginationRaw.(map[string]any)
	if !ok || pagination == nil {
		return "", ""
	}
	if v, ok := pagination["next_cursor"].(string); ok {
		next = v
	}
	if v, ok := pagination["previous_cursor"].(string); ok {
		prev = v
	}
	return next, prev
}

func (it *DataEngineIterator) buildCursorURL(cursor string) (*string, error) {
	if cursor == "" {
		return nil, nil
	}
	// Copy query for this page only so cursor is not left on initialQuery.
	params := make(Params, len(it.initialQuery)+1)
	for k, v := range it.initialQuery {
		params[k] = v
	}
	params["cursor"] = cursor

	session := it.resource.Session()
	fullURL, err := buildUrl(
		session,
		it.resource.GetResourcePath(),
		params.ToQuery(),
		session.GetConfig().ApiVersion,
		it.resource.GetApiRoot(),
	)
	if err != nil {
		return nil, err
	}
	return &fullURL, nil
}

func (it *DataEngineIterator) Next() (RecordSet, error) {
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

func (it *DataEngineIterator) Previous() (RecordSet, error) {
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

func (it *DataEngineIterator) HasNext() bool {
	if !it.initialized {
		return true
	}
	return it.nextURL != nil && *it.nextURL != ""
}

func (it *DataEngineIterator) HasPrevious() bool {
	if !it.initialized {
		return false
	}
	return it.previousURL != nil && *it.previousURL != ""
}

func (it *DataEngineIterator) Count() int    { return it.totalCount }
func (it *DataEngineIterator) PageSize() int { return it.pageSize }

func (it *DataEngineIterator) Reset() (RecordSet, error) {
	it.initialized = false
	it.current = nil
	it.nextURL = nil
	it.previousURL = nil
	it.currentPage = 0
	it.err = nil
	it.totalCount = -1
	return it.Next()
}

func (it *DataEngineIterator) All() (RecordSet, error) {
	var allRecords RecordSet
	if !it.initialized {
		records, err := it.Next()
		if err != nil {
			return nil, err
		}
		allRecords = append(allRecords, records...)
	} else {
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

func (it *DataEngineIterator) String() string {
	var sb strings.Builder
	sb.WriteString("DataEngineIterator {\n")
	sb.WriteString(fmt.Sprintf("  Initialized:   %v\n", it.initialized))
	sb.WriteString(fmt.Sprintf("  Current Page:  %d\n", it.currentPage))
	sb.WriteString(fmt.Sprintf("  Page Size:     %d\n", it.pageSize))
	sb.WriteString(fmt.Sprintf("  Total Count:   %d\n", it.totalCount))
	if len(it.current) > 0 {
		sb.WriteString(fmt.Sprintf("  Current:       [... (%d items)]\n", len(it.current)))
	} else {
		sb.WriteString("  Current:       []\n")
	}
	if it.nextURL != nil && *it.nextURL != "" {
		sb.WriteString(fmt.Sprintf("  Next URL:      %s\n", *it.nextURL))
	} else {
		sb.WriteString("  Next URL:      <none>\n")
	}
	if it.previousURL != nil && *it.previousURL != "" {
		sb.WriteString(fmt.Sprintf("  Previous URL:  %s\n", *it.previousURL))
	} else {
		sb.WriteString("  Previous URL:  <none>\n")
	}
	if it.err != nil {
		sb.WriteString(fmt.Sprintf("  Error:         %v\n", it.err))
	}
	sb.WriteString("}")
	return sb.String()
}
