package dto

import "time"

// Requests

type CreateBucketRequest struct {
	Name   string `json:"name"`
	Public bool   `json:"public"`
}

// Responses

type BucketResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Public    bool      `json:"public"`
	CreatedAt time.Time `json:"created_at"`
}

type BucketListResponse struct {
	Buckets []BucketResponse `json:"buckets"`
}
