package tinycast

// This is some terrible code.

// Page represents a page in a sequence of paginated results.
type Page struct {
	Num   int
	Title int
}

// Pagination represents a location in a sequences of paginated results.
type Pagination struct {
	numItems     int
	itemsPerPage int
	curPage      int
}

// NumPages returns the number of pages in the result set.
func (p Pagination) NumPages() int {
	if p.numItems == 0 {
		return 0
	}
	return (p.numItems + p.itemsPerPage - 1) / p.itemsPerPage
}

// CurrentPage returns the current page in the result set.
func (p Pagination) CurrentPage() int {
	return p.curPage
}

// NextPage returns the subsequent page in the result set.
func (p Pagination) NextPage() int {
	return p.curPage + 1
}

// PreviousPage returns the preceding page in the result set.
func (p Pagination) PreviousPage() int {
	return p.curPage - 1
}

// LastPage returnss the last page in the result set.
func (p Pagination) LastPage() int {
	return p.NumPages() - 1
}

// FirstItem returns the offset of the first item for the current page in the
// result set.
func (p Pagination) FirstItem() int {
	return p.itemsPerPage * p.curPage
}

// Pages returns a slide of Page elements.
func (p Pagination) Pages() []Page {
	pages := make([]Page, p.NumPages())
	for i := range pages {
		pages[i].Num = i
		pages[i].Title = i + 1
	}
	return pages
}
