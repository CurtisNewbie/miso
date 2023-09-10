package miso

import "gorm.io/gorm"

const (
	DEF_PAGE_LIMIT = 30
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
		p.Limit = DEF_PAGE_LIMIT
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

type QueryCondition[Req any] func(tx *gorm.DB, req Req) *gorm.DB
type BaseQuery func(tx *gorm.DB) *gorm.DB
type SelectQuery func(tx *gorm.DB) *gorm.DB
type QueryPageParam[T any, V any] struct {
	ReqPage         Paging            // Reques Paging Param
	Req             T                 // Request Object
	AddSelectQuery  SelectQuery       // Add SELECT query
	GetBaseQuery    BaseQuery         // Base query
	ApplyConditions QueryCondition[T] // Where Conditions
	ForEach         Peek[V]
}

func QueryPage[Req any, Res any](rail Rail, tx *gorm.DB, p QueryPageParam[Req, Res]) (PageRes[Res], error) {
	var res PageRes[Res]
	var total int

	// count
	t := p.ApplyConditions(p.GetBaseQuery(tx), p.Req).Select("COUNT(*)").Scan(&total)
	if t.Error != nil {
		return res, t.Error
	}

	var payload []Res

	// the actual page
	if total > 0 {
		t = p.AddSelectQuery(
			p.ApplyConditions(
				p.GetBaseQuery(tx),
				p.Req,
			),
		).Offset(p.ReqPage.GetOffset()).
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
