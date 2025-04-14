package model

type SubmitRequest struct {
	Code     string `json:"code" binding:"required"`
	Language string `json:"language" binding:"required"`
	Run      bool   `json:"run"`
	Stdin    string `json:"stdin"`
	BackEnd  string `json:"backend"`
}
