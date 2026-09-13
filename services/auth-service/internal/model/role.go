package model

type Role struct {
	Name        string
	IsAdmin     bool
	ShowHistory bool
	CanRead     bool
	CanDelete   bool
}
