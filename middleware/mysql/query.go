// deprecated: use middleware/dbquery package instead
package mysql

import (
	"github.com/curtisnewbie/miso/middleware/dbquery"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	"gorm.io/gorm"
)

type Query = dbquery.Query
type ChainedPageQuery = dbquery.ChainedPageQuery
type Nilable = dbquery.Nilable
type NilableValue = dbquery.NilableValue

var (
	NewQuery = dbquery.NewQuery
)

// Create param for page query.
//
// TODO: make this an alias of type dbquery.PageQuery in go1.24 (when generic type alias becomes available).
type PageQuery[V any] struct {
	db          *gorm.DB
	selectQuery ChainedPageQuery       // Add SELECT query and ORDER BY query, e.g., return tx.Select(`*`).
	baseQuery   ChainedPageQuery       // Base query, e.g., return tx.Table(`myTable`).Where(...)
	mapTo       util.Transform[V]      // callback triggered on each record, the value returned will overwrite the value passed in.
	mapToAsync  util.TransformAsync[V] // callback triggered on each record, the value returned will overwrite the value passed in.
}

func NewPagedQuery[V any](db *gorm.DB) *PageQuery[V] {
	return &PageQuery[V]{db: db}
}

func (pq *PageQuery[V]) WithSelectQuery(qry ChainedPageQuery) *PageQuery[V] {
	pq.selectQuery = qry
	return pq
}

func (pq *PageQuery[V]) WithBaseQuery(qry ChainedPageQuery) *PageQuery[V] {
	pq.baseQuery = qry
	return pq
}

func (pq *PageQuery[V]) Transform(t util.Transform[V]) *PageQuery[V] {
	pq.mapTo = t
	return pq
}

func (pq *PageQuery[V]) TransformAsync(t util.TransformAsync[V]) *PageQuery[V] {
	pq.mapToAsync = t
	return pq
}

func (pq *PageQuery[V]) Scan(rail miso.Rail, reqPage miso.Paging) (miso.PageRes[V], error) {
	newQuery := func() *Query {
		return pq.baseQuery(NewQuery(pq.db))
	}

	countFuture := util.RunAsync(func() (int, error) {
		var total int
		_, err := newQuery().Select("COUNT(*)").Scan(&total)
		return total, err
	})

	pageFuture := util.RunAsync(func() ([]V, error) {
		var payload []V
		qry := newQuery()
		if pq.selectQuery != nil {
			qry = pq.selectQuery(qry)
		}
		_, err := qry.Offset(reqPage.GetOffset()).
			Limit(reqPage.GetLimit()).
			Scan(&payload)
		if err != nil {
			return nil, err
		}

		if pq.mapTo != nil {
			for i := range payload {
				payload[i] = pq.mapTo(payload[i])
			}
		}

		if pq.mapToAsync != nil {
			futures := make([]util.Future[V], 0, len(payload))
			for _, p := range payload {
				futures = append(futures, pq.mapToAsync(p))
			}
			for i := range payload {
				v, err := futures[i].Get()
				if err != nil {
					rail.Warnf("Failed to resolve Future, skipped %v", err)
					continue
				}
				payload[i] = v
			}
		}
		return payload, nil
	})

	var res miso.PageRes[V]
	total, err := countFuture.Get()
	if err != nil {
		return res, err
	}

	payload, err := pageFuture.Get()
	if err != nil {
		return res, err
	}

	res = miso.PageRes[V]{Payload: payload, Page: miso.RespPage(reqPage, total)}
	return res, nil
}
