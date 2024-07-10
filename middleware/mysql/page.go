package mysql

import (
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	"gorm.io/gorm"
)

type PageQueryBuilder func(tx *gorm.DB) *gorm.DB

// Create param for page query.
type QueryPageParam[V any] struct {
	reqPage     miso.Paging       // Request Paging Param.
	selectQuery PageQueryBuilder  // Add SELECT query and ORDER BY query, e.g., return tx.Select(`*`).
	baseQuery   PageQueryBuilder  // Base query, e.g., return tx.Table(`myTable`).Where(...)
	forEach     util.Transform[V] // callback triggered on each record, the value returned will overwrite the value passed in.
}

func (q *QueryPageParam[V]) WithPage(p miso.Paging) *QueryPageParam[V] {
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
func (q *QueryPageParam[V]) Exec(rail miso.Rail, tx *gorm.DB) (miso.PageRes[V], error) {
	return QueryPage(rail, tx, *q)
}

// Execute paged query.
//
// COUNT query is called first, if none is found (i.e., COUNT(...) == 0), this method
// will not call the actual SELECT query to avoid unnecessary performance lost.
func QueryPage[Res any](rail miso.Rail, tx *gorm.DB, p QueryPageParam[Res]) (miso.PageRes[Res], error) {
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

	var res miso.PageRes[Res]
	total, err := countFuture.Get()
	if err != nil {
		return res, err
	}

	payload, err := pageFuture.Get()
	if err != nil {
		return res, err
	}

	res = miso.PageRes[Res]{Payload: payload, Page: miso.RespPage(p.reqPage, total)}
	return res, nil
}

func NewPageQuery[Res any]() *QueryPageParam[Res] {
	return new(QueryPageParam[Res])
}
