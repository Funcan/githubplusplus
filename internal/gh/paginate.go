package gh

import gogithub "github.com/google/go-github/v72/github"

// paginate calls fetch repeatedly, incrementing the page via resp.NextPage,
// until all pages have been collected. The first call uses page 0, which
// GitHub treats as page 1.
func paginate[T any](fetch func(page int) ([]T, *gogithub.Response, error)) ([]T, error) {
	var all []T
	page := 0
	for {
		items, resp, err := fetch(page)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}
	return all, nil
}
