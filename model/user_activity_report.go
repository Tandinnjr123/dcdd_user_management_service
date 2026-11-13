package model

import (
    "time"

    "github.com/google/uuid"
)

// UserActivityFilter narrows the DCDD user activity report. DCDD membership is a programme-wide
// plan and DCDD users don't redeem coupons, so login is the only per-user activity.
type UserActivityFilter struct {
    Activity string // all or login
    FromDate *time.Time
    ToDate   *time.Time
    SearchBy string
    Search   string
    Limit    int
    Offset   int
}

type UserActivityReportRow struct {
    UserID      uuid.UUID  `json:"user_id"`
    Name        *string    `json:"name"`
    Cid         *string    `json:"cid"`
    Email       *string    `json:"email"`
    MobileNo    *string    `json:"mobile_no"`
    Category    *string    `json:"category"`
    Status      *string    `json:"status"`
    LastLoginAt *time.Time `json:"last_login_at"`
}

type UserActivityReportResult struct {
    Users      []UserActivityReportRow `json:"users"`
    Total      int                     `json:"total"`
    Pagination *DcddUserPagination     `json:"pagination"`
}
