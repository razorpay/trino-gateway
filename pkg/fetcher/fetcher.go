package fetcher

const MaxLimit = 2000

// Request for fetching multiple entities
type FetchMultipleRequest struct {
	EntityName   string
	Filter       map[string]interface{}
	Pagination   Pagination
	TimeRange    TimeRange
	IsTrashed    bool
	HasCreatedAt bool
}

type IFetchMultipleRequest interface {
	GetEntityName() string
	GetFilter() map[string]interface{}
	GetPagination() IPagination
	GetTimeRange() ITimeRange
	GetTrashed() bool
	ContainsCreatedAt() bool
}

// GetEntityName : name of the entity to be fetched.
func (fr FetchMultipleRequest) GetEntityName() string {
	return fr.EntityName
}

// GetFilter : filter conditions for fetching the entities.
func (fr FetchMultipleRequest) GetFilter() map[string]interface{} {
	return fr.Filter
}

// GetPagination : skip and limit values for multiple fetch.
func (fr FetchMultipleRequest) GetPagination() IPagination {
	return fr.Pagination
}

// GetTimeRange : time range for multiple fetch query.
func (fr FetchMultipleRequest) GetTimeRange() ITimeRange {
	return fr.TimeRange
}

// GetTrashed : fetch soft deleted entities too if true.
func (fr FetchMultipleRequest) GetTrashed() bool {
	return fr.IsTrashed
}

// ContainsCreatedAt: account_balance table doesn't have create_at field in capital_loc..
func (fr FetchMultipleRequest) ContainsCreatedAt() bool {
	return fr.HasCreatedAt
}

// skip and limit values for multiple fetch.
type Pagination struct {
	Limit int
	Skip  int
}

type IPagination interface {
	GetLimit() int
	GetOffset() int
}

// GetLimit : get limit value for pagination.
func (p Pagination) GetLimit() int {
	if p.Limit > MaxLimit {
		return MaxLimit
	}
	return p.Limit
}

// GetOffset : get skip value for pagination.
func (p Pagination) GetOffset() int {
	return p.Skip
}

// Time range for the entity fetch request.
type TimeRange struct {
	From int64
	To   int64
}

type ITimeRange interface {
	GetTo() int64
	GetFrom() int64
}

// GetFrom : from timestamp.
func (tr TimeRange) GetFrom() int64 {
	return tr.From
}

// GetTo : to timestamp.
func (tr TimeRange) GetTo() int64 {
	return tr.To
}

// Response for multiple fetch request.
type FetchMultipleResponse struct {
	entities map[string]interface{}
}

type IFetchMultipleResponse interface {
	GetEntities() interface{}
}

// GetEntities : returns the fetched entities.
func (fr FetchMultipleResponse) GetEntities() interface{} {
	return fr.entities
}

// Single entity fetch request.
type FetchRequest struct {
	EntityName   string
	ID           string
	IsTrashed    bool
	HasCreatedAt bool
}

type IFetchRequest interface {
	GetID() string
	GetTrashed() bool
	GetEntityName() string
}

// GetID : Id of the entity to be fetched.
func (fr FetchRequest) GetID() string {
	return fr.ID
}

// GetEntityName : Name of the entity to be fetched.
func (fr FetchRequest) GetEntityName() string {
	return fr.EntityName
}

// GetTrashed : Include soft deleted records if true.
func (fr FetchRequest) GetTrashed() bool {
	return fr.IsTrashed
}

type FetchResponse struct {
	entity interface{}
}

// Response for single entity fetch
type IFetchResponse interface {
	GetEntity() interface{}
}

// Return fetched entity.
func (fr FetchResponse) GetEntity() interface{} {
	return fr.entity
}
