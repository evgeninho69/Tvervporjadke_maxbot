package dto

type CategoriesResponseAndRequest struct {
	Categories []CategoryResponseAndRequest `json:"categories"`
}

type CategoryResponseAndRequest struct {
	Name          string   `json:"name"`
	Subcategories []string `json:"subcategories"`
}
