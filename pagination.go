package tinycast

// This is some terrible code.

type Page struct {
	Num   int
	Title int
}

type Pagination struct {
	numItems     int
	itemsPerPage int
	curPage      int
}

func (p Pagination) NumPages() int {
	if p.numItems == 0 {
		return 0
	}
	return (p.numItems + p.itemsPerPage - 1) / p.itemsPerPage
}

func (p Pagination) CurrentPage() int {
	return p.curPage
}

func (p Pagination) NextPage() int {
	return p.curPage + 1
}

func (p Pagination) PreviousPage() int {
	return p.curPage - 1
}

func (p Pagination) LastPage() int {
	return p.NumPages() - 1
}

func (p Pagination) FirstItem() int {
	return p.itemsPerPage * p.curPage
}

func (p Pagination) Pages() []Page {
	pages := make([]Page, p.NumPages())

	for i, _ := range pages {
		pages[i].Num = i
		pages[i].Title = i + 1
	}
	return pages
}
