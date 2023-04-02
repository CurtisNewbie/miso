package common

type Paging struct {
	Limit int `json:"limit"`
	Page  int `json:"page"`
	Total int `json:"total"`
}

const (
	DEF_PAGE_LIMIT = 30
)

func (p Paging) GetOffset() int {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Limit < 1 {
		p.Limit = DEF_PAGE_LIMIT
	}
	return (p.Page - 1) * p.Limit
}

func (p Paging) GetLimit() int {
	if p.Limit < 1 {
		p.Limit = DEF_PAGE_LIMIT
	}
	return p.Limit
}

// Build Paging for response.
//
// deprecated, use RespPage() instead
func BuildResPage(reqPage *Paging, total int) *Paging {
	return &Paging{
		Limit: reqPage.Limit,
		Page:  reqPage.Page,
		Total: total,
	}
}

/* Build Paging for response */
func RespPage(reqPage Paging, total int) Paging {
	return Paging{
		Limit: reqPage.Limit,
		Page:  reqPage.Page,
		Total: total,
	}
}

// Calculate offset.
//
// deprecated, use Paging.GetOffset() instead
func CalcOffset(paging *Paging) int {
	if paging.Page < 1 {
		paging.Page = 1
	}
	if paging.Limit < 1 {
		paging.Limit = 30
	}
	return (paging.Page - 1) * paging.Limit
}
