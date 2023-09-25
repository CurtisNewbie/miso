package miso

import "gorm.io/gorm"

const (
	DefaultPageLimit = 30
)

type Paging struct {
	Limit int `json:"limit"`
	Page  int `json:"page"`
	Total int `json:"total"`
}

type PageRes[T any] struct {
	Page    Paging `json:"pagingVo"`
	Payload []T    `json:"payload"`
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
	ReqPage         Paging           // Request Paging Param.
	AddSelectQuery  PageQueryBuilder // Add SELECT query, e.g., return tx.Select(`*`).
	GetBaseQuery    PageQueryBuilder // Base query, e.g., return tx.Table(`myTable`).
	ApplyConditions PageQueryBuilder // Where Conditions, optional, e.g., return tx.Where(`field = 'abc'`).
	ForEach         Transform[V]     // callback triggered on each record, the value returned will overwrite the value passed in.
}

// Execute paged query
//
//	This internally calls QueryPage(...).
func (q QueryPageParam[V]) ExecPageQuery(rail Rail, tx *gorm.DB) (PageRes[V], error) {
	return QueryPage(rail, tx, q)
}

// Execute paged query.
//
// COUNT query is called first, if none is found (i.e., COUNT(...) == 0), this method will not call the actual SELECT query to avoid unnecessary performance lost.
func QueryPage[Res any](rail Rail, tx *gorm.DB, p QueryPageParam[Res]) (PageRes[Res], error) {
	var res PageRes[Res]
	var total int

	newQuery := func() *gorm.DB {
		t := p.GetBaseQuery(tx)
		if p.ApplyConditions != nil {
			t = p.ApplyConditions(t)
		}
		return t
	}

	// count
	t := newQuery().Select("COUNT(*)").Scan(&total)

	if t.Error != nil {
		return res, t.Error
	}

	var payload []Res

	// the actual page
	if total > 0 {
		t = p.AddSelectQuery(newQuery()).
			Offset(p.ReqPage.GetOffset()).
			Limit(p.ReqPage.GetLimit()).
			Scan(&payload)
		if t.Error != nil {
			return res, t.Error
		}

		if p.ForEach != nil {
			for i := range payload {
				payload[i] = p.ForEach(payload[i])
			}
		}
	}

	return PageRes[Res]{Payload: payload, Page: RespPage(p.ReqPage, total)}, nil
}
