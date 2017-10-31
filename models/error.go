package models

const (
	ErrorNotFound = iota
	ErrorAlreadyExists
)

//easyjson:json
type Error struct {
	Type  uint	   `json:"-"`
	Message string `json:"message"`
}