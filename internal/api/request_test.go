package api

import "testing"

func TestReviewRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		req     ReviewRequest
		max     int
		wantErr bool
	}{
		{name: "valid", req: ReviewRequest{Title: "Add auth", Diff: "diff --git"}, max: 50000, wantErr: false},
		{name: "missing title", req: ReviewRequest{Diff: "diff --git"}, max: 50000, wantErr: true},
		{name: "missing diff", req: ReviewRequest{Title: "Add auth"}, max: 50000, wantErr: true},
		{name: "large diff", req: ReviewRequest{Title: "Add auth", Diff: "123456"}, max: 5, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate(tt.max)
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}
