package miso

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

func (p *Paging) NextPage() {
	p.Page += 1
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
