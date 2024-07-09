package mysql

import (
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	"gorm.io/gorm"
)

const (
	DefaultPageLimit = 30
)

type Paging struct {
	Limit int `json:"limit" desc:"page limit"`
	Page  int `json:"page" desc:"page number, 1-based"`
	Total int `json:"total" desc:"total count"`
}

type PageRes[T any] struct {
	Page    Paging `json:"paging" desc:"pagination parameters"`
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
	reqPage     Paging            // Request Paging Param.
	selectQuery PageQueryBuilder  // Add SELECT query and ORDER BY query, e.g., return tx.Select(`*`).
	baseQuery   PageQueryBuilder  // Base query, e.g., return tx.Table(`myTable`).Where(...)
	forEach     util.Transform[V] // callback triggered on each record, the value returned will overwrite the value passed in.
}

func (q *QueryPageParam[V]) WithPage(p Paging) *QueryPageParam[V] {
	q.reqPage = p
	return q
}

func (q *QueryPageParam[V]) WithSelectQuery(qry PageQueryBuilder) *QueryPageParam[V] {
	q.selectQuery = qry
	return q
}

func (q *QueryPageParam[V]) WithBaseQuery(qry PageQueryBuilder) *QueryPageParam[V] {
	q.baseQuery = qry
	return q
}

func (q *QueryPageParam[V]) ForEach(t util.Transform[V]) *QueryPageParam[V] {
	q.forEach = t
	return q
}

// Execute paging query
func (q *QueryPageParam[V]) Exec(rail miso.Rail, tx *gorm.DB) (PageRes[V], error) {
	return QueryPage(rail, tx, *q)
}

// Execute paged query.
//
// COUNT query is called first, if none is found (i.e., COUNT(...) == 0), this method
// will not call the actual SELECT query to avoid unnecessary performance lost.
func QueryPage[Res any](rail miso.Rail, tx *gorm.DB, p QueryPageParam[Res]) (PageRes[Res], error) {
	newQuery := func() *gorm.DB {
		return p.baseQuery(tx)
	}

	countFuture := util.RunAsync(func() (int, error) {
		var total int
		t := newQuery().Select("COUNT(*)").Scan(&total)
		return total, t.Error
	})
	pageFuture := util.RunAsync(func() ([]Res, error) {
		var payload []Res
		t := newQuery()
		if p.selectQuery != nil {
			t = p.selectQuery(t)
		}
		t = t.Offset(p.reqPage.GetOffset()).
			Limit(p.reqPage.GetLimit()).
			Scan(&payload)
		if t.Error != nil {
			return nil, t.Error
		}

		if p.forEach != nil {
			for i := range payload {
				payload[i] = p.forEach(payload[i])
			}
		}
		return payload, nil
	})

	var res PageRes[Res]
	total, err := countFuture.Get()
	if err != nil {
		return res, err
	}

	payload, err := pageFuture.Get()
	if err != nil {
		return res, err
	}

	res = PageRes[Res]{Payload: payload, Page: RespPage(p.reqPage, total)}
	return res, nil
}

func NewPageQuery[Res any]() *QueryPageParam[Res] {
	return new(QueryPageParam[Res])
}
