package common

type Paging struct {
	Limit int `json:"limit"`
	Page  int `json:"page"`
	Total int `json:"total"`
}

/* Build Paging for response (deprecated, use RespPage() instead) */
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

// Calculate offset
func CalcOffset(paging *Paging) int {
	if paging.Page < 1 {
		paging.Page = 1
	}
	if paging.Limit < 1 {
		paging.Limit = 30
	}
	return (paging.Page - 1) * paging.Limit
}
