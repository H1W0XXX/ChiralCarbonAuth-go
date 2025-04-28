// File: types.go
package main

// Challenge holds data for a captcha challenge
type Challenge struct {
	Regions []string // 所有可选区域，比如 ["A1","A2",...]
	Answers []string // 正确答案区域列表
}

// StartResponse is returned by /api/challenge/start
type StartResponse struct {
	UUID    string   `json:"uuid"`
	Image   string   `json:"image"`   // Base64 PNG
	Regions []string `json:"regions"` // 全部可选区域
}

// VerifyRequest is the JSON body for /api/challenge/verify
type VerifyRequest struct {
	UUID       string   `json:"uuid"`
	Selections []string `json:"selections"`
}

// VerifyResponse is returned by /api/challenge/verify
type VerifyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
