package entity

type Category struct {
	Name string
	Subcategories []string
}

type Categories struct {
	Categories []Category
}