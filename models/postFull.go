package models

//easyjson:json
type PostFull struct {
	Author *User   `json:"author,omitempty"`
	Forum *Forum	`json:"forum,omitempty"`
	Post *Post	`json:"post,omitempty"`
	Thread *Thread `json:"thread,omitempty"`
}
