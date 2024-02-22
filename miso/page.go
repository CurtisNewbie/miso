package miso

import "gorm.io/gorm"

const (
	DefaultPageLimit = 30
)

type Paging struct {
	Limit int `json:"limit" desc:"page limit"`
	Page  int `json:"page" desc:"page number, 1-based"`
	Total int `json:"total" desc:"total count"`
}

type PageRes[T any] struct {
	Page    Paging `json:"pagingVo" desc:"pagination parameters"`
	Payload []T    `json:"payload" desc:"payload values in current page"`
}

func (p Paging) GetPage() int {
	if p.Page < 1 {
		return 1
	}
	return p.Page
}

func (p Paging) GetOffset() int {
	return (p.GetPage() - 1) * p.GetLimit()
}

func (p Paging) GetLimit() int {
	if p.Limit < 1 {
		p.Limit = DefaultPageLimit
	}
	return p.Limit
}

func (p Paging) ToRespPage(total int) Paging {
	return RespPage(p, total)
}

/* Build Paging for response */
func RespPage(reqPage Paging, total int) Paging {
	return Paging{
		Limit: reqPage.GetLimit(),
		Page:  reqPage.GetPage(),
		Total: total,
	}
}

type PageQueryBuilder func(tx *gorm.DB) *gorm.DB

// Create param for page query.
type QueryPageParam[V any] struct {
	ReqPage     Paging           // Request Paging Param.
	SelectQuery PageQueryBuilder // Add SELECT query and ORDER BY query, e.g., return tx.Select(`*`).
	BaseQuery   PageQueryBuilder // Base query, e.g., return tx.Table(`myTable`).Where(...)
	DoForEach   Transform[V]     // callback triggered on each record, the value returned will overwrite the value passed in.
}

func (q *QueryPageParam[V]) WithPage(p Paging) *QueryPageParam[V] {
	q.ReqPage = p
	return q
}

func (q *QueryPageParam[V]) WithSelectQuery(qry PageQueryBuilder) *QueryPageParam[V] {
	q.SelectQuery = qry
	return q
}

func (q *QueryPageParam[V]) WithBaseQuery(qry PageQueryBuilder) *QueryPageParam[V] {
	q.BaseQuery = qry
	return q
}

func (q *QueryPageParam[V]) ForEach(t Transform[V]) *QueryPageParam[V] {
	q.DoForEach = t
	return q
}

// Execute paging query
func (q *QueryPageParam[V]) Exec(rail Rail, tx *gorm.DB) (PageRes[V], error) {
	return QueryPage(rail, tx, *q)
}

// Execute paged query.
//
// COUNT query is called first, if none is found (i.e., COUNT(...) == 0), this method will not call the actual SELECT query to avoid unnecessary performance lost.
func QueryPage[Res any](rail Rail, tx *gorm.DB, p QueryPageParam[Res]) (PageRes[Res], error) {
	var res PageRes[Res]
	var total int

	newQuery := func() *gorm.DB {
		return p.BaseQuery(tx)
	}

	// count
	t := newQuery().Select("COUNT(*)").Scan(&total)

	if t.Error != nil {
		return res, t.Error
	}

	var payload []Res

	// the actual page
	if total > 0 {
		t = p.SelectQuery(newQuery()).
			Offset(p.ReqPage.GetOffset()).
			Limit(p.ReqPage.GetLimit()).
			Scan(&payload)
		if t.Error != nil {
			return res, t.Error
		}

		if p.DoForEach != nil {
			for i := range payload {
				payload[i] = p.DoForEach(payload[i])
			}
		}
	}

	return PageRes[Res]{Payload: payload, Page: RespPage(p.ReqPage, total)}, nil
}

func NewPageQuery[Res any]() *QueryPageParam[Res] {
	return new(QueryPageParam[Res])
}
